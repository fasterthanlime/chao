package main

import (
	"bufio"
	"context"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqliteutil"
	"github.com/go-xorm/builder"
)

//===================================================
// Workarounds
//===================================================
var (
	// This fixes it completely for me (all conns are healthy at the end)
	neverInterruptQueries = false

	// This doesn't seem to fix anything
	useTransient = false

	// This lets you attach with a debugger before we touch connections
	facilitateDebugging = false
)

//===================================================
// Simulation parameters (shouldn't need to touch)
//===================================================
var (
	// size of connection pool
	poolSize = 10

	// in milliseconds
	deadlineBase int64 = 100
	// in milliseconds
	deadlineVariance int64 = 100

	// number of records to upate in each step
	// THIS IS ADJUSTED AUTOMATICALLY depending on
	// how long queries take
	numUpdates = 4 * 1000

	// number of goroutines trying to read & write from/to DB
	numWorkers = 15
	// number of transactions each worker will attempt to do
	numSteps = 3

	// number of times we'll spin up {numWorkers} goroutines
	// (we sleep in between)
	numRounds = 3

	// how many seconds we wait between each round
	recessSeconds = 2
)

type ExecFunc = func(conn *sqlite.Conn, query string, resultFn func(stmt *sqlite.Stmt) error, args ...interface{}) error

var Exec ExecFunc

func init() {
	if useTransient {
		Exec = sqliteutil.ExecTransient
	} else {
		Exec = sqliteutil.Exec
	}
}

func main() {
	log.SetFlags(0)
	pool, err := sqlite.Open("file:memory:?mode=memory", 0, poolSize)
	must(err)

	migrate := func() {
		migConn := pool.Get(context.Background().Done())
		if migConn == nil {
			panic("could not get conn for migration")
		}

		err := sqliteutil.Exec(migConn, "CREATE TABLE thieves (id INTEGER PRIMARY KEY, name TEXT)", nil)
		must(err)

		pool.Put(migConn)
	}
	migrate()

	names := []string{
		"Tess", "Daniel", "Rusty", "Linus", "Saul", "Terry", "Basher",
	}

	var step func(n int, prng *rand.Rand)
	step = func(n int, prng *rand.Rand) {
		startTime := time.Now()
		deadline := time.Millisecond * time.Duration(int64(deadlineBase)+prng.Int63n(deadlineVariance))
		var conn *sqlite.Conn

		defer func() {
			if r := recover(); r != nil {
				duration := time.Since(startTime)
				if err, ok := r.(error); ok {
					errCode := sqlite.ErrCode(err)
					switch errCode {
					case sqlite.SQLITE_INTERRUPT:
						if duration < deadline {
							advance := deadline - duration
							log.Printf("%3d XXXX [%p] interrupted %s before deadline (%s duration)", n, conn, advance, duration)
						} else {
							lateness := duration - deadline
							log.Printf("%3d .... [%p] interrupted %s after deadline (%s duration)", n, conn, lateness, duration)
							if numUpdates > 4000 {
								numUpdates = int(float64(numUpdates) * 0.9)
								log.Printf("(-) numUpdates = %d", numUpdates)
							}
						}
						return
					case sqlite.SQLITE_LOCKED, sqlite.SQLITE_LOCKED_SHAREDCACHE:
						const maxLockTries = 60
						if n > maxLockTries {
							log.Printf("%3d ---- [%p] was locked, giving up after %d tries", n, conn, maxLockTries)
							return
						}
						lockSleep := time.Duration(100) * time.Millisecond
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
					advance := deadline - duration
					log.Printf("%3d ...- [%p] succeeded %s early", n, conn, advance)
					if advance > 10*time.Millisecond {
						numUpdates = int(float64(numUpdates) * 1.2)
						log.Printf("(+) numUpdates = %d", numUpdates)
					}
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

		if neverInterruptQueries {
			// This prevents any calls to C.sqlite3_interrupt()
			conn.SetInterrupt(context.Background().Done())
		}

		err := func() (err error) {
			defer sqliteutil.Save(conn)(&err)

			{
				ids := make([]interface{}, 400)
				id := int64(0)
				for i := range ids {
					id += prng.Int63n(10)
					ids = append(ids, int64(i))
				}
				sql, args, err := builder.Select().From("thieves").Where(builder.In("id", ids...)).ToSQL()
				if err != nil {
					return err
				}
				err = sqliteutil.Exec(conn, sql, nil, args...)
				if err != nil {
					return err
				}
			}

			{
				id := int64(0)
				for i := 0; i < numUpdates; i++ {
					id += prng.Int63n(10)

					name := names[prng.Intn(len(names))]
					sql := `INSERT INTO thieves
				(id, name) VALUES (?, ?)
				ON CONFLICT (id) DO UPDATE
				SET name=excluded.name`
					err = sqliteutil.Exec(conn, sql, nil, id, name)
					if err != nil {
						return err
					}
				}
			}
			return nil
		}()
		must(err)
	}

	done := make(chan bool)
	worker := func() {
		prng := rand.New(rand.NewSource(time.Now().UnixNano()))
		for i := 0; i < numSteps; i++ {
			step(0, prng)
			// time.Sleep(time.Duration(prng.Int63n(600)) * time.Millisecond)
		}
	}

	for rounds := 0; rounds < numRounds; rounds++ {
		globalStartTime := time.Now()

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

		log.Printf("Sleeping %d seconds to let SQLite settle...", recessSeconds)
		for i := 1; i <= recessSeconds; i++ {
			log.Printf("%s", strings.Repeat(".", i))
			time.Sleep(1 * time.Second)
		}
	}

	if facilitateDebugging {
		log.Printf("You can attach to me with:")
		log.Printf("dlv attach %d", os.Getpid())
		log.Printf("(Press Enter to continue - also execute 'continue' in dlv)")

		reader := bufio.NewReader(os.Stdin)
		reader.ReadLine()
	}

	log.Printf("Now testing connections...")
	healthyConns := 0
	for i := 0; i < poolSize; i++ {
		func(i int) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			conn := pool.Get(ctx.Done())
			if conn == nil {
				log.Printf("[%d] got nil conn", i)
				return
			}
			log.Printf("Testing connection %d...", i)
			err := sqliteutil.Exec(conn, "SELECT * FROM thieves LIMIT 1", nil)
			if err != nil {
				log.Printf("[%d] %+v", i, err)
				return
			}

			healthyConns++
		}(i)
		// Onto the next conn... (not putting conn back to make sure we cycle through all conns)
	}

	log.Printf("Found %d / %d healthy conns", healthyConns, poolSize)

	if healthyConns < poolSize {
		log.Printf(" :( ")
	} else {
		log.Printf(" :) ")
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
