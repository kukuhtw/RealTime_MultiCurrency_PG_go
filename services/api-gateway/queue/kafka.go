// services/api-gateway/queue/kafka.go
package queue

import (
    "context"
    "time"

    "github.com/segmentio/kafka-go"
)

type Bus struct {
    Brokers      []string
    RequestTopic string
    ResultTopic  string
}

func New(brokers []string, reqTopic, resTopic string) *Bus {
    return &Bus{Brokers: brokers, RequestTopic: reqTopic, ResultTopic: resTopic}
}

func (b *Bus) Publish(ctx context.Context, key, payload []byte) error {
    w := &kafka.Writer{
        Addr:     kafka.TCP(b.Brokers...),
        Topic:    b.RequestTopic,
        Balancer: &kafka.LeastBytes{},
    }
    defer w.Close()
    return w.WriteMessages(ctx, kafka.Message{Key: key, Value: payload})
}

// WaitResult: tunggu message di topic result dg key yg sama
func (b *Bus) WaitResult(ctx context.Context, key []byte, timeout time.Duration) (value []byte, ok bool, err error) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  b.Brokers,
        Topic:    b.ResultTopic,
        GroupID:  "", // stateless reader (tiap request)
        MinBytes: 1,
        MaxBytes: 10e6,
    })
    defer r.Close()

    for {
        m, e := r.ReadMessage(ctx)
        if e != nil {
            return nil, false, e
        }
        if string(m.Key) == string(key) {
            return m.Value, true, nil
        }
    }
}
