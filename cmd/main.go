package main

import (
	"context"
	"os"

	"github.com/keptn-contrib/dynatrace-service/internal/common"
	"github.com/keptn-contrib/dynatrace-service/internal/credentials"
	"github.com/keptn-contrib/dynatrace-service/internal/event_handler"
	"github.com/keptn-contrib/dynatrace-service/internal/lib"
	"github.com/keptn-contrib/dynatrace-service/internal/onboard"

	log "github.com/sirupsen/logrus"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/kelseyhightower/envconfig"
)

type envConfig struct {
	// Port on which to listen for cloudevents
	Port int    `envconfig:"RCV_PORT" default:"8080"`
	Path string `envconfig:"RCV_PATH" default:"/"`
}

func main() {
	log.SetLevel(lib.GetLogLevel())

	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.WithError(err).Fatal("Failed to process env var")
	}

	if common.RunLocal || common.RunLocalTest {
		log.Info("env=runlocal: Running with local filesystem to fetch resources")
	}

	os.Exit(_main(os.Args[1:], env))
}

func _main(args []string, env envConfig) int {

	if lib.IsServiceSyncEnabled() {
		cm, err := credentials.NewCredentialManager(nil)
		if err != nil {
			log.WithError(err).Fatal("Failed to initialize CredentialManager")
		}
		onboard.ActivateServiceSynchronizer(cm)
	}

	ctx := context.Background()
	ctx = cloudevents.WithEncodingStructured(ctx)

	p, err := cloudevents.NewHTTP(cloudevents.WithPath(env.Path), cloudevents.WithPort(env.Port))
	if err != nil {
		log.WithError(err).Fatal("Failed to create client")
	}
	c, err := cloudevents.NewClient(p)
	if err != nil {
		log.WithError(err).Fatal("Failed to create client")
	}
	log.Fatal(c.StartReceiver(ctx, gotEvent))

	return 0
}

func gotEvent(ctx context.Context, event cloudevents.Event) error {

	dynatraceEventHandler, err := event_handler.NewEventHandler(event)

	if err != nil {
		log.WithError(err).Error("NewEventHandler() returned an error")
		return err
	}

	err = dynatraceEventHandler.HandleEvent()
	if err != nil {
		log.WithError(err).Error("HandleEvent() returned an error")
	}
	return err
}
