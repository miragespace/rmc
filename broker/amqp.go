package broker

import (
	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec"

	extErrors "github.com/pkg/errors"
	"github.com/streadway/amqp"
	"google.golang.org/protobuf/proto"
)

var _ Broker = &AMQPBroker{}

const workerRequestExchange string = "worker_controls"

// AMQPBroker describes a message broker via RabbitMQ
type AMQPBroker struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

// NewAMQPBroker returns a Message Broker over RabbitMQ
func NewAMQPBroker(amqpURI string) (Broker, error) {
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
	if err := broker.setupExchange(); err != nil {
		return nil, extErrors.Wrap(err, "Cannot declare exchange for worker controls")
	}

	return broker, nil
}

func (a *AMQPBroker) setupExchange() error {
	return a.channel.ExchangeDeclare(
		workerRequestExchange, // name
		"direct",              // type
		true,                  // durable
		false,                 // auto-deleted
		false,                 // internal
		false,                 // no-wait
		nil,                   // arguments
	)
}

// Close will close the channel and connection to release resources
func (a *AMQPBroker) Close() {
	a.channel.Close()
	a.connection.Close()
}

func (a *AMQPBroker) publishViaRoutingKey(routingKey string, body []byte) error {
	return a.channel.Publish(
		workerRequestExchange,
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
func (a *AMQPBroker) SendControlRequest(host *host.Host, p *spec.ControlRequest) error {
	protoBytes, err := proto.Marshal(p)
	if err != nil {
		return extErrors.Wrap(err, "Cannot encode message into bytes")
	}
	if err := a.publishViaRoutingKey(host.Identifier(), protoBytes); err != nil {
		return extErrors.Wrap(err, "Cannot publish control request")
	}
	return nil
}

// SendProvisionRequest will send request to provision to a specific host
func (a *AMQPBroker) SendProvisionRequest(host *host.Host, p *spec.ProvisionRequest) error {
	protoBytes, err := proto.Marshal(p)
	if err != nil {
		return extErrors.Wrap(err, "Cannot encode message into bytes")
	}
	if err := a.publishViaRoutingKey(host.Identifier(), protoBytes); err != nil {
		return extErrors.Wrap(err, "Cannot publish provision request")
	}
	return nil
}
