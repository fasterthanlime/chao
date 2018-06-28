package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"crawshaw.io/sqlite"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type Human struct {
	ID   int64
	Name string
}

func main() {
	consumer := &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			log.Printf("[%s] %s", lvl, msg)
		},
	}
	models := []interface{}{
		&Human{},
	}

	c, err := hades.NewContext(consumer, models...)
	must(err)

	pool, err := sqlite.Open("database.db", 0, 10)
	must(err)

	conn := pool.Get(context.Background().Done())
	defer pool.Put(conn)

	err = c.AutoMigrate(conn)
	must(err)

	names := []string{
		"Tess", "Daniel", "Rusty", "Linus", "Saul", "Terry", "Basher",
	}

	var step func(n int, prng *rand.Rand)
	step = func(n int, prng *rand.Rand) {
		startTime := time.Now()
		deadline := time.Duration(100*(20+10*prng.Int63n(2))) * time.Millisecond

		defer func() {
			if r := recover(); r != nil {
				duration := time.Since(startTime)
				if err, ok := r.(error); ok {
					errCode := sqlite.ErrCode(errors.Cause(err))
					switch errCode {
					case sqlite.SQLITE_INTERRUPT:
						if duration < deadline {
							log.Printf("%d ⏰⏰⏰ we got interrupted %s before the deadline (%s duration)", n, deadline-duration, duration)
						} else {
							log.Printf("%d 🛑 %s after deadline (%s duration)", n, duration-deadline, duration)
						}
						return
					case sqlite.SQLITE_LOCKED:
						log.Printf("%d 🔒 locked", n)
						lockSleep := time.Duration(50+prng.Int63n(50)) * time.Millisecond
						time.Sleep(lockSleep)
						step(n+1, prng)
						return
					default:
						log.Printf("%d 🔫 a new challenger appears: %+v", n, errCode)
						return
					}
				}
				log.Printf("%d 🔥 %s / %s: %s", n, duration, deadline, r)
			} else {
				duration := time.Since(startTime)
				if duration >= deadline {
					log.Printf("%d 👍 ⚠️  %s late", n, duration-deadline)
				} else {
					log.Printf("%d 👍 %s early", n, deadline-duration)
				}
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), deadline)
		defer cancel()

		conn := pool.Get(ctx.Done())
		if conn == nil {
			panic("we got a nil conn :o")
		}
		defer pool.Put(conn)

		records := make([]*Human, 2*1000)
		id := prng.Int63n(256 * 1024)

		for i := range records {
			records[i] = &Human{
				ID:   id,
				Name: names[prng.Intn(len(names))],
			}
			id += prng.Int63n(256)
		}

		must(c.Save(conn, records))
	}

	numSteps := 3
	done := make(chan bool)
	worker := func() {
		prng := rand.New(rand.NewSource(time.Now().UnixNano()))
		for i := 0; i < numSteps; i++ {
			step(0, prng)
		}
	}

	globalStartTime := time.Now()

	numWorkers := 100
	log.Printf("Spinning up %d workers doing %d steps...", numWorkers, numSteps)
	for i := 0; i < numWorkers; i++ {
		go func() {
			worker()
			done <- true
		}()
	}

	for i := 0; i < numWorkers; i++ {
		<-done
		log.Printf("Worker joined...")
	}

	globalDuration := time.Since(globalStartTime)
	log.Printf("All done in %s", globalDuration)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
