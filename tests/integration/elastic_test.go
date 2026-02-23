package integration

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	elasticclient "github.com/elastic/go-elasticsearch/v8"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	adapterses "github.com/nimafallahian/go-workflow/internal/adapters/es"
	"github.com/nimafallahian/go-workflow/internal/domain"
)

func TestElasticsearchIndexerIndexesMessageEvent(t *testing.T) {
	// If Docker is not available, skip Elasticsearch integration tests.
	if _, err := os.Stat("/var/run/docker.sock"); err != nil {
		t.Skip("docker not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        "docker.elastic.co/elasticsearch/elasticsearch:8.9.0",
		ExposedPorts: []string{"9200/tcp"},
		Env: map[string]string{
			"discovery.type":        "single-node",
			"xpack.security.enabled": "false",
		},
		WaitingFor: wait.ForHTTP("/_cluster/health").
			WithPort("9200/tcp").
			WithStartupTimeout(90 * time.Second),
	}

	esContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer func() { _ = esContainer.Terminate(context.Background()) }()

	endpoint, err := esContainer.PortEndpoint(ctx, "9200", "http")
	require.NoError(t, err)

	clientCfg := elasticclient.Config{
		Addresses: []string{endpoint},
	}
	es, err := elasticclient.NewClient(clientCfg)
	require.NoError(t, err)

	indexer, err := adapterses.NewIndexer(es, "messages")
	require.NoError(t, err)

	event := domain.MessageEvent{
		ID: "msg-es-1",
		Payload: map[string]any{
			"foo": "bar",
		},
		Metadata: map[string]string{
			"source": "integration-test",
		},
		StatusCode: 200,
	}

	err = indexer.Index(ctx, []domain.MessageEvent{event})
	require.NoError(t, err)

	// Wait a brief moment; refresh=wait_for should already have made the doc visible.
	time.Sleep(2 * time.Second)

	res, err := es.Get("messages", event.ID, es.Get.WithContext(ctx))
	require.NoError(t, err)
	defer res.Body.Close()

	require.False(t, res.IsError(), "expected successful get, got status %s", res.Status())

	var body struct {
		Found  bool            `json:"found"`
		Source json.RawMessage `json:"_source"`
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	require.NoError(t, err)
	require.True(t, body.Found, "expected document to be found")
}


