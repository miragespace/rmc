package broker

import (
	"context"

	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec/broker"
	"github.com/zllovesuki/rmc/spec/protocol"

	extErrors "github.com/pkg/errors"
	"github.com/streadway/amqp"
	"google.golang.org/protobuf/proto"
)

var _ broker.Producer = &AMQPBroker{}
var _ broker.Consumer = &AMQPBroker{}

const (
	workerControlExchange   string = "worker_control"
	workerProvisionExchange        = "worker_provision"
)

// AMQPBroker describes a message broker via RabbitMQ
type AMQPBroker struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

// NewAMQPBroker returns a Message Broker over RabbitMQ
func NewAMQPBroker(amqpURI string) (*AMQPBroker, error) {
	amqpConn, err := amqp.Dial(amqpURI)
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot connect to Message Broker")
	}
	amqpChan, err := amqpConn.Channel()
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot create broken channel")
	}
	broker := &AMQPBroker{
		connection: amqpConn,
		channel:    amqpChan,
	}
	if err := broker.setupControlExchange(); err != nil {
		return nil, extErrors.Wrap(err, "Cannot declare exchange for control requests")
	}
	if err := broker.setupProvisionExchange(); err != nil {
		return nil, extErrors.Wrap(err, "Cannot declare exchange for provision requests")
	}

	return broker, nil
}

func (a *AMQPBroker) setupControlExchange() error {
	return a.channel.ExchangeDeclare(
		workerControlExchange, // name
		"direct",              // type
		true,                  // durable
		false,                 // auto-deleted
		false,                 // internal
		false,                 // no-wait
		nil,                   // arguments
	)
}

func (a *AMQPBroker) setupProvisionExchange() error {
	return a.channel.ExchangeDeclare(
		workerProvisionExchange, // name
		"direct",                // type
		true,                    // durable
		false,                   // auto-deleted
		false,                   // internal
		false,                   // no-wait
		nil,                     // arguments
	)
}

// Close will close the channel and connection to release resources
func (a *AMQPBroker) Close() {
	a.channel.Close()
	a.connection.Close()
}

func (a *AMQPBroker) publishViaRoutingKey(exchange, routingKey string, body []byte) error {
	return a.channel.Publish(
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
func (a *AMQPBroker) SendControlRequest(host *host.Host, p *protocol.ControlRequest) error {
	protoBytes, err := proto.Marshal(p)
	if err != nil {
		return extErrors.Wrap(err, "Cannot encode message into bytes")
	}
	if err := a.publishViaRoutingKey(workerControlExchange, host.Identifier(), protoBytes); err != nil {
		return extErrors.Wrap(err, "Cannot publish control request")
	}
	return nil
}

// SendProvisionRequest will send request to provision to a specific host
func (a *AMQPBroker) SendProvisionRequest(host *host.Host, p *protocol.ProvisionRequest) error {
	protoBytes, err := proto.Marshal(p)
	if err != nil {
		return extErrors.Wrap(err, "Cannot encode message into bytes")
	}
	if err := a.publishViaRoutingKey(workerProvisionExchange, host.Identifier(), protoBytes); err != nil {
		return extErrors.Wrap(err, "Cannot publish provision request")
	}
	return nil
}

func (a *AMQPBroker) setupQueue(qName string) error {
	_, err := a.channel.QueueDeclare(
		qName,
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}

func (a *AMQPBroker) bindAndGetMsgChan(qName, exchange string, host *host.Host) (<-chan amqp.Delivery, error) {
	if err := a.channel.QueueBind(
		qName,
		host.Identifier(),
		exchange,
		false,
		nil,
	); err != nil {
		return nil, err
	}
	msgChan, err := a.channel.Consume(
		qName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	return msgChan, err
}

func (a *AMQPBroker) ReceiveControlRequest(ctx context.Context, host *host.Host) (<-chan *protocol.ControlRequest, error) {
	name := "control_" + host.Identifier()
	if err := a.setupQueue(name); err != nil {
		return nil, extErrors.Wrap(err, "Cannot setup queue")
	}
	msgChan, err := a.bindAndGetMsgChan(name, workerControlExchange, host)
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

func (a *AMQPBroker) ReceiveProvisionRequest(ctx context.Context, host *host.Host) (<-chan *protocol.ProvisionRequest, error) {
	name := "provision_" + host.Identifier()
	if err := a.setupQueue(name); err != nil {
		return nil, extErrors.Wrap(err, "Cannot setup queue")
	}
	msgChan, err := a.bindAndGetMsgChan(name, workerProvisionExchange, host)
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

func (a *AMQPBroker) Heartbeart(host *host.Host, b *protocol.Heartbeat) error {
	return nil
}
