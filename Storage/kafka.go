package Kafka

import (
	"github.com/Shopify/sarama"
	"watchever.com/CLog"
)

const (
	_KAFKA_COMMON_HOST_          = "kafka.vme-tech.com"
	_KAFKA_COMMON_CONSUMER_PORT_ = 2181
	_KAFKA_COMMON_PRODUCER_PORT_ = 9092
	_KAFKA_TOPIC_                = "playerlog"
)

func Test(json string) {
	producerList := [1]string{fmt.Sprintf("%s:%d", _KAFKA_COMMON_HOST_, _KAFKA_COMMON_CONSUMER_PORT_)}

	producer, err := NewAsyncProducer(producerList, nil)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := producer.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	// Trap SIGINT to trigger a shutdown.
	// signals := make(chan os.Signal, 1)
	// signal.Notify(signals, os.Interrupt)

	var enqueued, errors int
ProducerLoop:
	for {
		select {
		case producer.Input() <- &ProducerMessage{Topic: _KAFKA_TOPIC_, Key: nil, Value: StringEncoder(json)}:
			enqueued++
		case err := <-producer.Errors():
			log.Println("Failed to produce message", err)
			errors++
		case <-signals:
			break ProducerLoop
		}
	}

	clog.Trace("Kafka", "Test", "Enqueued: %d; errors: %d", enqueued, errors)
}
