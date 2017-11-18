package storage

// import (
// 	"fmt"
// 	"log"
//
// 	"github.com/Shopify/sarama"
// )
//
// const (
// 	_KAFKA_COMMON_HOST_          = "kafka.vme-tech.com"
// 	_KAFKA_COMMON_CONSUMER_PORT_ = 2181
// 	_KAFKA_COMMON_PRODUCER_PORT_ = 9092
// 	_KAFKA_TOPIC_                = "test_go"
// )
//
// type Driver struct {
// 	ConnString []string
// 	Store      interface{}
// }
//
// func Init() *Driver {
// 	var err error
//
// 	d := &Driver{
// 		ConnString: []string{fmt.Sprintf("%s:%d", _KAFKA_COMMON_HOST_, _KAFKA_COMMON_PRODUCER_PORT_)},
// 	}
//
// 	d.Store, err = sarama.NewAsyncProducer(d.ConnString, nil)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	return d
// }
//
// // ./kafka-console-consumer.sh --zookeeper localhost:2181 --topic test_go
// func (d *Driver) NewRecord(json string) {
// 	defer func() {
// 		if err := d.Store.(sarama.AsyncProducer).Close(); err != nil {
// 			log.Fatalln(err)
// 		}
// 	}()
//
// 	select {
// 	case d.Store.(sarama.AsyncProducer).Input() <- &sarama.ProducerMessage{Topic: _KAFKA_TOPIC_, Key: nil, Value: sarama.StringEncoder(json)}:
// 	case err := <-d.Store.(sarama.AsyncProducer).Errors():
// 		log.Println("Failed to produce message", err)
// 	}
// }
