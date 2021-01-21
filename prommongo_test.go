package prommongo_test

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/johejo/prommongo"
)

func Test(t *testing.T) {
	cmc := prommongo.NewCommandMonitorCollector()
	pmc := prommongo.NewPoolMonitorCollector()
	reg := prometheus.NewRegistry()
	reg.MustRegister(cmc, pmc)

	opts := options.Client().
		ApplyURI("mongodb://root:example@localhost:27017").
		SetMonitor(cmc.CommandMonitor(nil)).
		SetPoolMonitor(pmc.PoolMonitor(nil))
	client, err := mongo.NewClient(opts)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	db := client.Database("testing")
	coll := db.Collection("prommongo")
	defer func() {
		if err := db.Drop(ctx); err != nil {
			t.Fatal(err)
		}
		if err := client.Disconnect(ctx); err != nil {
			t.Fatal(err)
		}
	}()
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		value := "Gopher-" + strconv.Itoa(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := coll.InsertOne(ctx, bson.M{"name": value})
			if err != nil {
				panic(err)
			}

			var got map[string]interface{}
			if err := coll.FindOne(ctx, bson.M{"name": value}).Decode(&got); err != nil {
				panic(err)
			}
			_ = got
		}()
	}

	ms, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	names := []string{
		"go_mongo_command_duration_ns",
		"go_mongo_connection_closed",
		"go_mongo_connection_created",
		"go_mongo_connection_returnd",
		"go_mongo_get_failed",
		"go_mongo_get_succeeded",
		"go_mongo_pool_cleared",
		"go_mongo_pool_closed",
		"go_mongo_pool_created",
		"go_mongo_max_pool_size",
		"go_mongo_min_pool_size",
		"go_mongo_wait_queue_timeout_ms",
	}
	type result struct {
		found bool
	}
	results := make(map[string]result)
	for _, name := range names {
		results[name] = result{found: false}
	}
	for _, m := range ms {
		for _, name := range names {
			if *m.Name == name {
				results[name] = result{found: true}
				break
			}
		}
	}

	for name, result := range results {
		if !result.found {
			t.Errorf("%s not found", name)
		}
	}
}
