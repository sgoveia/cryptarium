package correlate_test

import (
	"testing"

	"github.com/sgoveia/cryptarium/internal/correlate"
	"github.com/sgoveia/cryptarium/internal/model"
)

func TestLink_PrimitiveParameters(t *testing.T) {
	findings := []model.CryptoFinding{
		{
			ID: "src", Primitive: "RSA", Parameters: map[string]any{"keySize": 2048},
			Evidence: model.Evidence{Source: model.SourceCode, Path: "token.go", Line: 9, Confidence: model.ConfidenceHigh},
		},
		{
			ID: "cert", Primitive: "RSA", Parameters: map[string]any{"keySize": 2048},
			Evidence: model.Evidence{Source: model.SourceCertificate, Path: "certs/rsa.pem", Confidence: model.ConfidenceHigh},
		},
		{
			ID: "deps", Primitive: "RSA", Parameters: map[string]any{"module": "golang.org/x/crypto"},
			Evidence: model.Evidence{Source: model.SourceDependency, Path: "go.mod", Line: 6, Confidence: model.ConfidenceMedium},
		},
		{
			ID: "ecdsa", Primitive: "ECDSA", Parameters: map[string]any{"curve": "P-256"},
			Evidence: model.Evidence{Source: model.SourceCertificate, Path: "certs/ec.pem", Confidence: model.ConfidenceHigh},
		},
	}
	clusters := correlate.Link(findings)
	if len(clusters) != 2 {
		t.Fatalf("clusters=%d want 2", len(clusters))
	}
	var rsaCluster correlate.Cluster
	for _, c := range clusters {
		if len(c.Members) == 3 {
			rsaCluster = c
		}
	}
	if len(rsaCluster.Members) != 3 {
		t.Fatalf("RSA cluster members=%d", len(rsaCluster.Members))
	}
	asset := correlate.Materialize(rsaCluster)
	if asset.Evidence.Source != model.SourceCode {
		t.Fatalf("canonical source=%s want source-code", asset.Evidence.Source)
	}
	if len(asset.RelatedIDs) != 2 {
		t.Fatalf("RelatedIDs=%v", asset.RelatedIDs)
	}
	if asset.Evidence.Confidence != model.ConfidenceHigh {
		t.Fatalf("confidence=%s want high after source+deps link", asset.Evidence.Confidence)
	}
	if asset.QuantumClass != model.ClassBroken {
		t.Fatalf("class=%s", asset.QuantumClass)
	}
}

func TestLink_PathProximityCertConfig(t *testing.T) {
	findings := []model.CryptoFinding{
		{
			ID: "c", Primitive: "RSA", Parameters: map[string]any{"keySize": 2048},
			Evidence: model.Evidence{Source: model.SourceCertificate, Path: "deploy/api.pem"},
		},
		{
			ID: "cfg", Primitive: "RSA",
			Evidence: model.Evidence{Source: model.SourceConfiguration, Path: "deploy/nginx.conf", Line: 7},
		},
	}
	clusters := correlate.Link(findings)
	if len(clusters) != 1 || len(clusters[0].Members) != 2 {
		t.Fatalf("got %+v", clusters)
	}
}

func TestLink_Deterministic(t *testing.T) {
	findings := []model.CryptoFinding{
		{ID: "b", Primitive: "AES", Parameters: map[string]any{"keySize": 128}, Evidence: model.Evidence{Path: "b.conf", Source: model.SourceConfiguration}},
		{ID: "a", Primitive: "AES", Parameters: map[string]any{"keySize": 128}, Evidence: model.Evidence{Path: "a.conf", Source: model.SourceConfiguration}},
	}
	c1 := correlate.Link(findings)
	c2 := correlate.Link(findings)
	if len(c1) != 1 || len(c2) != 1 {
		t.Fatal(c1, c2)
	}
	if c1[0].Members[0].ID != c2[0].Members[0].ID {
		t.Fatalf("order mismatch")
	}
}
