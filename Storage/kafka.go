package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/Djoulzy/Polycom/CLog"
	"github.com/Shopify/sarama"
)

const (
	_KAFKA_COMMON_HOST_          = "kafka.vme-tech.com"
	_KAFKA_COMMON_CONSUMER_PORT_ = 2181
	_KAFKA_COMMON_PRODUCER_PORT_ = 9092
	_KAFKA_TOPIC_                = "playerlog"
)

func Test(json string) {
	producerList := []string{fmt.Sprintf("%s:%d", _KAFKA_COMMON_HOST_, _KAFKA_COMMON_PRODUCER_PORT_)}

	producer, err := sarama.NewAsyncProducer(producerList, nil)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := producer.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	// Trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	var enqueued, errors int
ProducerLoop:
	for {
		select {
		case producer.Input() <- &sarama.ProducerMessage{Topic: _KAFKA_TOPIC_, Key: nil, Value: sarama.StringEncoder(json)}:
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

func main() {
	clog.LogLevel = 5
	clog.StartLogging = true

	message := "{\"SID\":\"VM\",\"TCPADDR\":\"10.31.100.200:8081\",\"HTTPADDR\":\"10.31.100.200:8080\",\"HOST\":\"HTTP: 10.31.100.200:8080 - TCP: 10.31.100.200:8081\",\"CPU\":2,\"GORTNE\":8,\"STTME\":\"12/06/2017 11:45:18\",\"UPTME\":\"25.00085595s\",\"LSTUPDT\":\"12/06/2017 11:45:43\",\"LAVG\":5,\"MEM\":\"\u003cth\u003eMem\u003c/th\u003e\u003ctd class='memCell'\u003e3963 Mo\u003c/td\u003e\u003ctd class='memCell'\u003e3653 Mo\u003c/td\u003e\u003ctd class='memCell'\u003e6.8%\u003c/td\u003e\",\"SWAP\":\"\u003cth\u003eSwap\u003c/th\u003e\u003ctd class='memCell'\u003e1707 Mo\u003c/td\u003e\u003ctd class='memCell'\u003e1707 Mo\u003c/td\u003e\u003ctd class='memCell'\u003e0.0%\u003c/td\u003e\",\"NBMESS\":1,\"NBI\":0,\"MXI\":500,\"NBU\":0,\"MXU\":200,\"NBM\":0,\"MXM\":3,\"NBS\":1,\"MXS\":5,\"BRTHLST\":{\"iMac\":{\"Tcpaddr\":\"10.31.200.168:8081\",\"Httpaddr\":\"localhost:8080\"}}}"
	Test(message)
}
