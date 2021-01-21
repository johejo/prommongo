package prommongo_test

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/johejo/prommongo"
)

func Example() {
	cmc := prommongo.NewCommandMonitorCollector()
	pmc := prommongo.NewPoolMonitorCollector()
	reg := prometheus.NewRegistry()
	reg.MustRegister(cmc, pmc)

	client, err := mongo.NewClient(
		options.Client().
			ApplyURI("mongodb://root:example@localhost:27017").
			SetMonitor(cmc.CommandMonitor(nil)).
			SetPoolMonitor(pmc.PoolMonitor(nil)),
	)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		panic(err)
	}
	db := client.Database("testing")
	coll := db.Collection("prommongo")
	defer func() {
		if err := db.Drop(ctx); err != nil {
			panic(err)
		}
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	_, err = coll.InsertOne(ctx, bson.M{"name": "Gopher"})
	if err != nil {
		panic(err)
	}

	if err := coll.FindOne(ctx, bson.M{"name": "Gopher"}).Err(); err != nil {
		panic(err)
	}

	ms, err := reg.Gather()
	if err != nil {
		panic(err)
	}
	for _, m := range ms {
		fmt.Println(*m.Name)
	}

	// Output:
	// go_mongo_command_duration_ns
	// go_mongo_connection_closed
	// go_mongo_connection_created
	// go_mongo_connection_returnd
	// go_mongo_get_failed
	// go_mongo_get_succeeded
	// go_mongo_max_pool_size
	// go_mongo_min_pool_size
	// go_mongo_pool_cleared
	// go_mongo_pool_closed
	// go_mongo_pool_created
	// go_mongo_wait_queue_timeout_ms
}
