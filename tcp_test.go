package tcp

import (
	"fmt"
	"io/ioutil"
	"net"
	"testing"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/codec/format"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/elastic/beats/libbeat/outputs/outest"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/stretchr/testify/assert"
)

var config = Config{
	Host: "127.0.0.1",
	Port: "9000",
	lineDelimiter: "\n",
}

func TestRunServer(t *testing.T) {
	server()
}

func server() {
	ln, err := net.Listen(networkTCP, net.JoinHostPort(config.Host, config.Port))
	if err != nil {
		fmt.Println("start server err: ", err)
		return
	}
	defer ln.Close()
	for {
		c, _ := ln.Accept()
		go func() {
			for {
				buf := make([]byte, 1 << 15)
				_, err := c.Read(buf)
				if err != nil {
					c.Close()
					return
				}
			}
		}()
	}
}

func newBenchmarkBatch(size int) *outest.Batch {
	var events []beat.Event
	for i := 0; i < size; i++ {
		events = append(events, beat.Event{
			Fields: event("field", "value"),
		})
	}
	return outest.NewBatch(events...)
}

func BenchmarkBufio(b *testing.B) {
	batch := newBenchmarkBatch(100)
	enc := json.New(false, true, "1.2.3")
	config.WritevEnable = false
	t, err := newTcpOut("test", config, outputs.NewNilObserver(), enc)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		t.Publish(batch)
	}
	t.Close()
}

func BenchmarkWritev(b *testing.B) {
	batch := newBenchmarkBatch(100)
	enc := json.New(false, true, "1.2.3")
	config.WritevEnable = true
	t, err := newTcpOut("test", config, outputs.NewNilObserver(), enc)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		t.Publish(batch)
	}
	t.Close()
}


func TestTcpOutput(t *testing.T) {
	var tests = []struct {
		title    string
		codec    codec.Codec
		events   []beat.Event
		expected string
	}{
		{
			"single json event (pretty=false)",
			json.New(false, true, "1.2.3"),
			[]beat.Event{
				{Fields: event("field", "value")},
			},
			"{\"@timestamp\":\"0001-01-01T00:00:00.000Z\",\"@metadata\":{\"beat\":\"test\",\"type\":\"doc\",\"version\":\"1.2.3\"},\"field\":\"value\"}\n",
		},
		{
			"single json event (pretty=true)",
			json.New(true, true, "1.2.3"),
			[]beat.Event{
				{Fields: event("field", "value")},
			},
			"{\n  \"@timestamp\": \"0001-01-01T00:00:00.000Z\",\n  \"@metadata\": {\n    \"beat\": \"test\",\n    \"type\": \"doc\",\n    \"version\": \"1.2.3\"\n  },\n  \"field\": \"value\"\n}\n",
		},
		{
			"event with custom format string",
			format.New(fmtstr.MustCompileEvent("%{[event]}")),
			[]beat.Event{
				{Fields: event("event", "myevent")},
			},
			"myevent\n",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.title, func(t *testing.T) {
			batch := outest.NewBatch(test.events...)
			lines, err := run(config, test.codec, batch)
			assert.Nil(t, err)
			assert.Equal(t, test.expected, lines)

			// check batch correctly signalled
			if !assert.Len(t, batch.Signals, 1) {
				return
			}
			assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
		})
	}
}

func withTcpOut(ln net.Listener, fn func()) (string, error) {
	outC, errC := make(chan string), make(chan error)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			errC <- err
			return
		}
		b, err := ioutil.ReadAll(c)
		if err != nil {
			errC <- err
			return
		}
		outC <- string(b)
	}()

	fn()

	select {
	case result := <- outC:
		return result, nil
	case err := <- errC:
		return "", err
	}
}

func run(config Config, codec codec.Codec, batches ...publisher.Batch) (string, error) {
	ln, err := net.Listen(networkTCP, net.JoinHostPort(config.Host, config.Port))
	if err != nil {
		return "", err
	}
	defer ln.Close()

	t, err := newTcpOut("test", config, outputs.NewNilObserver(), codec)
	if err != nil {
		return "", err
	}
	return withTcpOut(ln, func() {
		for _, b := range batches {
			t.Publish(b)
		}
		t.Close()
	})
}

func event(k, v string) common.MapStr {
	return common.MapStr{k: v}
}
