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

	poolSize := 2
	// pool, err := sqlite.Open("database.db", 0, poolSize)
	pool, err := sqlite.Open("file:memory:?mode=memory", 0, poolSize)
	must(err)

	migConn := pool.Get(context.Background().Done())
	err = c.AutoMigrate(migConn)
	pool.Put(migConn)
	must(err)

	names := []string{
		"Tess", "Daniel", "Rusty", "Linus", "Saul", "Terry", "Basher",
	}

	log.SetFlags(0)

	var step func(n int, prng *rand.Rand)
	step = func(n int, prng *rand.Rand) {
		startTime := time.Now()
		// deadline := time.Duration(100*(8+3*prng.Int63n(2))) * time.Millisecond
		deadline := 300 * time.Millisecond
		var conn *sqlite.Conn

		defer func() {
			if r := recover(); r != nil {
				duration := time.Since(startTime)
				if err, ok := r.(error); ok {
					errCode := sqlite.ErrCode(errors.Cause(err))
					switch errCode {
					case sqlite.SQLITE_INTERRUPT:
						if duration < deadline {
							log.Printf("%3d XXXX [%p] interrupted %s before deadline (%s duration)", n, conn, deadline-duration, duration)
							// log.Printf("interrupt stack: %+v", err)
						} else {
							log.Printf("%3d .... [%p] interrupted %s after deadline (%s duration)", n, conn, duration-deadline, duration)
						}
						return
					case sqlite.SQLITE_LOCKED:
						log.Printf("%3d .... [%p] locked", n, conn)
						// log.Printf("full stack: %+v", err)
						// os.Exit(1)
						lockSleep := time.Duration(50+prng.Int63n(50)) * time.Millisecond
						time.Sleep(lockSleep)
						step(n+1, prng)
						return
					case sqlite.SQLITE_MISUSE:
						log.Printf("%3d !!!! [%p] misuse", n, conn)
						return
					default:
						log.Printf("%3d ???? [%p] a new challenger appears: %+v", n, conn, err)
						return
					}
				}
				log.Printf("%3d .... [%p] %s (%s / %s)", n, conn, r, duration, deadline)
			} else {
				duration := time.Since(startTime)
				if duration >= deadline {
					log.Printf("%3d ...+ [%p] succeeded %s late", n, conn, duration-deadline)
				} else {
					log.Printf("%3d ...- [%p] succeeded %s early", n, conn, deadline-duration)
				}
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), deadline)
		defer cancel()

		conn = pool.Get(ctx.Done())
		if conn == nil {
			panic("pool timeout")
		}
		defer pool.Put(conn)

		records := make([]*Human, 3*1000)
		// id := prng.Int63n(256 * 1024)
		id := int64(0)

		for i := range records {
			records[i] = &Human{
				ID:   id,
				Name: names[prng.Intn(len(names))],
			}
			// id += prng.Int63n(256)
			id += 1
		}

		must(c.Save(conn, records))
	}

	numSteps := 5
	done := make(chan bool)
	worker := func() {
		prng := rand.New(rand.NewSource(time.Now().UnixNano()))
		for i := 0; i < numSteps; i++ {
			step(0, prng)
		}
	}

	rounds := 3
	for rounds > 0 {
		rounds--
		globalStartTime := time.Now()

		numWorkers := 20
		log.Printf("Spinning up %d workers doing %d steps...", numWorkers, numSteps)
		for i := 0; i < numWorkers; i++ {
			go func() {
				worker()
				done <- true
			}()
		}

		for i := 0; i < numWorkers; i++ {
			<-done
		}

		globalDuration := time.Since(globalStartTime)
		log.Printf("All done in %s", globalDuration)

		time.Sleep(1 * time.Second)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
