package broker

import (
	"context"
	"fmt"

	"github.com/zllovesuki/rmc/spec/broker"
	"github.com/zllovesuki/rmc/spec/protocol"

	extErrors "github.com/pkg/errors"
	"github.com/streadway/amqp"
	"google.golang.org/protobuf/proto"
)

var _ broker.Producer = &AMQPBroker{}
var _ broker.Consumer = &AMQPBroker{}

const (
	instanceControlExchange   string = "host_control_exchange"
	instanceProvisionExchange        = "host_provision_exchange"
	hostReplyRoutingKey              = "request_reply"
	hostHeartbeatExchange            = "heartbeat"
)

// AMQPBroker describes a message broker via RabbitMQ
type AMQPBroker struct {
	connection      *amqp.Connection
	producerChannel *amqp.Channel
	consumerChannel *amqp.Channel
}

// NewAMQPBroker returns a Message Broker over RabbitMQ
func NewAMQPBroker(amqpURI string) (*AMQPBroker, error) {
	amqpConn, err := amqp.Dial(amqpURI)
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot connect to Message Broker")
	}
	return &AMQPBroker{
		connection: amqpConn,
	}, nil
}

// Producer will establish a channel to broker and returns a Producer
func (a *AMQPBroker) Producer() (broker.Producer, error) {
	var err error
	a.producerChannel, err = a.connection.Channel()
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot create producer channel")
	}
	if err := a.setupExchanges(); err != nil {
		return nil, extErrors.Wrap(err, "Cannot declare as Producer")
	}
	return a, nil
}

// Consumer will establish a channel to broker and returns a Consumer
func (a *AMQPBroker) Consumer() (broker.Consumer, error) {
	var err error
	a.consumerChannel, err = a.connection.Channel()
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot create consumer channel")
	}
	if err := a.setupExchanges(); err != nil {
		return nil, extErrors.Wrap(err, "Cannot declare as Consumer")
	}
	return a, nil
}

func (a *AMQPBroker) setupExchanges() error {
	exchanges := []string{
		instanceControlExchange,
		instanceProvisionExchange,
		hostHeartbeatExchange,
	}
	var channel *amqp.Channel
	if a.producerChannel != nil {
		channel = a.producerChannel
	}
	if a.consumerChannel != nil {
		channel = a.consumerChannel
	}
	if channel == nil {
		return fmt.Errorf("No open channel to setup exchanges")
	}
	for _, exchange := range exchanges {
		if err := channel.ExchangeDeclare(
			exchange, // name
			"topic",  // type
			true,     // durable
			false,    // auto-deleted
			false,    // internal
			false,    // no-wait
			nil,      // arguments
		); err != nil {
			return extErrors.Wrap(err, "Cannot setup exchange")
		}
	}
	return nil
}

// Close will close the channel and connection to release resources
func (a *AMQPBroker) Close() {
	if a.producerChannel != nil {
		a.producerChannel.Close()
	}
	if a.consumerChannel != nil {
		a.consumerChannel.Close()
	}
	a.connection.Close()
}

func (a *AMQPBroker) publishViaRoutingKey(exchange, routingKey string, body []byte) error {
	return a.producerChannel.Publish(
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/x-protobuf",
			Body:        body,
		},
	)
}

// SendControlRequest will send the request to control to a specific host
func (a *AMQPBroker) SendControlRequest(hostIdentifier string, p *protocol.ControlRequest) error {
	protoBytes, err := proto.Marshal(p)
	if err != nil {
		return extErrors.Wrap(err, "Cannot encode message into bytes")
	}
	if err := a.publishViaRoutingKey(instanceControlExchange, hostIdentifier, protoBytes); err != nil {
		return extErrors.Wrap(err, "Cannot publish control request")
	}
	return nil
}

// SendProvisionRequest will send request to provision to a specific host
func (a *AMQPBroker) SendProvisionRequest(hostIdentifier string, p *protocol.ProvisionRequest) error {
	protoBytes, err := proto.Marshal(p)
	if err != nil {
		return extErrors.Wrap(err, "Cannot encode message into bytes")
	}
	if err := a.publishViaRoutingKey(instanceProvisionExchange, hostIdentifier, protoBytes); err != nil {
		return extErrors.Wrap(err, "Cannot publish provision request")
	}
	return nil
}

// SendControlReply will send the control result back to the producer
func (a *AMQPBroker) SendControlReply(p *protocol.ControlReply) error {
	protoBytes, err := proto.Marshal(p)
	if err != nil {
		return extErrors.Wrap(err, "Cannot encode message into bytes")
	}
	if err := a.publishViaRoutingKey(instanceControlExchange, hostReplyRoutingKey, protoBytes); err != nil {
		return extErrors.Wrap(err, "Cannot publish control reply")
	}
	return nil
}

// SendProvisionReply will send the provision result back to the producer
func (a *AMQPBroker) SendProvisionReply(p *protocol.ProvisionReply) error {
	protoBytes, err := proto.Marshal(p)
	if err != nil {
		return extErrors.Wrap(err, "Cannot encode message into bytes")
	}
	if err := a.publishViaRoutingKey(instanceProvisionExchange, hostReplyRoutingKey, protoBytes); err != nil {
		return extErrors.Wrap(err, "Cannot publish provision reply")
	}
	return nil
}

func (a *AMQPBroker) getMsgChannel(qName, exchange, routingKey string) (<-chan amqp.Delivery, error) {
	if _, err := a.consumerChannel.QueueDeclare(
		qName, // name
		true,  // durable
		false, // auto delete
		false, // exclusive
		false, // no wait
		nil,   // args
	); err != nil {
		return nil, err
	}
	if err := a.consumerChannel.QueueBind(
		qName,      // name
		routingKey, // routing key
		exchange,   // exchange
		false,      // no wait
		nil,        // args
	); err != nil {
		return nil, err
	}
	msgChan, err := a.consumerChannel.Consume(
		qName, // queue
		"",    // consumer tag
		false, // auto ack
		false, // exclusive
		false, // no local
		false, // no wait
		nil,   // args
	)
	return msgChan, err
}

// ReceiveControlRequest will consumer control requests directed to the host
func (a *AMQPBroker) ReceiveControlRequest(ctx context.Context, hostIdentifier string) (<-chan *protocol.ControlRequest, error) {
	name := "control_" + hostIdentifier
	msgChan, err := a.getMsgChannel(name, instanceControlExchange, hostIdentifier)
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot setup consumer")
	}
	rChan := make(chan *protocol.ControlRequest)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d := <-msgChan:
				var req protocol.ControlRequest
				if err := proto.Unmarshal(d.Body, &req); err != nil {
					d.Nack(false, false)
					continue
				}
				rChan <- &req
				d.Ack(false)
			}
		}
	}()
	return rChan, nil
}

// ReceiveProvisionRequest will consumer provision requests directed to the host
func (a *AMQPBroker) ReceiveProvisionRequest(ctx context.Context, hostIdentifier string) (<-chan *protocol.ProvisionRequest, error) {
	name := "provision_" + hostIdentifier
	msgChan, err := a.getMsgChannel(name, instanceProvisionExchange, hostIdentifier)
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot setup consumer")
	}
	rChan := make(chan *protocol.ProvisionRequest)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d := <-msgChan:
				var req protocol.ProvisionRequest
				if err := proto.Unmarshal(d.Body, &req); err != nil {
					d.Nack(false, false)
					continue
				}
				rChan <- &req
				d.Ack(false)
			}
		}
	}()
	return rChan, nil
}

// ReceiveControlReply will consumer control replies from hosts
func (a *AMQPBroker) ReceiveControlReply(ctx context.Context) (<-chan *protocol.ControlReply, error) {
	name := "process_control_" + hostReplyRoutingKey
	msgChan, err := a.getMsgChannel(name, instanceControlExchange, hostReplyRoutingKey)
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot setup consumer")
	}
	rChan := make(chan *protocol.ControlReply)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d := <-msgChan:
				var req protocol.ControlReply
				if err := proto.Unmarshal(d.Body, &req); err != nil {
					d.Nack(false, false)
					continue
				}
				rChan <- &req
				d.Ack(false)
			}
		}
	}()
	return rChan, nil
}

// ReceiveProvisionReply will consumer provision replies from hosts
func (a *AMQPBroker) ReceiveProvisionReply(ctx context.Context) (<-chan *protocol.ProvisionReply, error) {
	name := "process_provision_" + hostReplyRoutingKey
	msgChan, err := a.getMsgChannel(name, instanceProvisionExchange, hostReplyRoutingKey)
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot setup consumer")
	}
	rChan := make(chan *protocol.ProvisionReply)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d := <-msgChan:
				var req protocol.ProvisionReply
				if err := proto.Unmarshal(d.Body, &req); err != nil {
					d.Nack(false, false)
					continue
				}
				rChan <- &req
				d.Ack(false)
			}
		}
	}()
	return rChan, nil
}

// SendHeartbeat signals the host is alive along with host metadata
func (a *AMQPBroker) SendHeartbeat(b *protocol.Heartbeat) error {
	protoBytes, err := proto.Marshal(b)
	if err != nil {
		return extErrors.Wrap(err, "Cannot encode message into bytes")
	}
	if err := a.publishViaRoutingKey(hostHeartbeatExchange, "heartbeat", protoBytes); err != nil {
		return extErrors.Wrap(err, "Cannot publish heartbeats")
	}
	return nil
}

// ReceiveHeartbeat will consumer heartbeats from hosts
func (a *AMQPBroker) ReceiveHeartbeat(ctx context.Context) (<-chan *protocol.Heartbeat, error) {
	name := "process_" + hostHeartbeatExchange
	msgChan, err := a.getMsgChannel(name, hostHeartbeatExchange, "#")
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot setup consumer")
	}
	hChan := make(chan *protocol.Heartbeat)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d := <-msgChan:
				var req protocol.Heartbeat
				if err := proto.Unmarshal(d.Body, &req); err != nil {
					d.Nack(false, false)
					continue
				}
				hChan <- &req
				d.Ack(false)
			}
		}
	}()
	return hChan, nil
}
