package heartbeat

import (
	"log"
	"time"

	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/controlplane"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/policystore"
)

type Loop struct {
	client           *controlplane.Client
	store            *policystore.Store
	interval         time.Duration
	listenerStatus   map[string]string
	certStatus       map[string]string
	managePublicCert bool
}

func New(
	client *controlplane.Client,
	store *policystore.Store,
	interval time.Duration,
	listenerStatus map[string]string,
	certStatus map[string]string,
	managePublicCert bool,
) *Loop {
	return &Loop{
		client:           client,
		store:            store,
		interval:         interval,
		listenerStatus:   cloneMap(listenerStatus),
		certStatus:       cloneMap(certStatus),
		managePublicCert: managePublicCert,
	}
}

func (l *Loop) Run() {
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()
	l.tick()
	for range ticker.C {
		l.tick()
	}
}

func (l *Loop) tick() {
	policy, err := l.client.FetchPolicy()
	if err != nil {
		log.Printf("fetch policy failed: %v", err)
		return
	}
	if err := l.store.Update(policy.PolicyRevisionID, policy.PayloadJSON); err != nil {
		log.Printf("update policy failed: %v", err)
		return
	}
	if l.managePublicCert {
		result, renewErr := l.client.RenewCertificate("public")
		if renewErr != nil {
			log.Printf("public cert renew failed: %v", renewErr)
			l.certStatus["public"] = "degraded"
		} else {
			l.certStatus["public"] = result.Status
		}
	}
	revision, _ := l.store.Current()
	_, err = l.client.SendHeartbeat(revision, cloneMap(l.listenerStatus), cloneMap(l.certStatus))
	if err != nil {
		log.Printf("heartbeat failed: %v", err)
	}
}

func cloneMap(value map[string]string) map[string]string {
	cloned := make(map[string]string, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}
