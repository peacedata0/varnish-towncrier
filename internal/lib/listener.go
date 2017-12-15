package lib

import (
	"io"
	"log"
	"time"

	"github.com/garyburd/redigo/redis"
)

type Listener struct {
	Options Options
}

func NewListener(options Options) *Listener {
	l := Listener{}
	l.Options = options

	return &l
}

func (l *Listener) Listen() error {

	var dialOptions []redis.DialOption
	{
		redis.DialConnectTimeout(5 * time.Second)
		redis.DialReadTimeout(2 * time.Second)
		redis.DialWriteTimeout(2 * time.Second)
	}

	if l.Options.Redis.Password != "" {
		dialOptions = append(dialOptions, redis.DialPassword(l.Options.Redis.Password))
	}

	rp := NewRequestProcessor(l.Options)

	for {
		log.Printf("Connecting to redis...")

		c, err := redis.DialURL(l.Options.Redis.Uri, dialOptions...)

		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		defer c.Close()

		log.Printf("Connected to %s", l.Options.Redis.Uri)

		psc := redis.PubSubConn{Conn: c}
		psc.Subscribe(redis.Args{}.AddFlat(l.Options.Redis.Subscribe)...)

	Receive:
		for {
			switch v := psc.Receive().(type) {
			case redis.Message:
				go rp.Process(string(v.Data))
			case redis.Subscription:
				log.Printf("%s: %s (%d)\n", v.Kind, v.Channel, v.Count)
			case error:
				if c.Err() == io.EOF {
					log.Print("Lost connection to redis, reconnecting...")
				} else {
					log.Print(c.Err())
					log.Print(v)
				}

				time.Sleep(5 * time.Second)
				c.Close()
				break Receive
			}
		}
	}

}