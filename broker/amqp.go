package broker

import (
	extErrors "github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec"
)

var _ Broker = &AMQPBroker{}

// AMQPBroker describes a message broker via RabbitMQ
type AMQPBroker struct {
	connection *amqp.Connection
	Channel    *amqp.Channel
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
	return &AMQPBroker{
		connection: amqpConn,
		Channel:    amqpChan,
	}, nil
}

func (a *AMQPBroker) Close() {
	a.Channel.Close()
	a.connection.Close()
}

func (a *AMQPBroker) SendControlRequest(host *host.Host, p *spec.ControlRequest) error {
	panic("not implemented")
}

func (a *AMQPBroker) SendProvisionRequest(host *host.Host, p *spec.ProvisionRequest) error {
	panic("not implemented")
}
