package src

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	MaxGoroutines = 2
	NetTimeout    = 2 * time.Second
	PrintTimeout  = time.Second
)

type SiteStatus struct {
	Name          string
	StatusCode    int
	TimeOfRequest time.Time
}

type Monitor struct {
	StatusMap        map[string]SiteStatus
	Mtx              *sync.Mutex
	G                errgroup.Group
	Sites            []string
	RequestFrequency time.Duration
}

func NewMonitor(sites []string, requestFrequency time.Duration) *Monitor {
	return &Monitor{
		StatusMap:        make(map[string]SiteStatus),
		Mtx:              &sync.Mutex{},
		Sites:            sites,
		RequestFrequency: requestFrequency,
	}
}

func (m *Monitor) Run(ctx context.Context) error {
	m.G.SetLimit(MaxGoroutines)

	m.G.Go(func() error {
		ticker := time.NewTicker(m.RequestFrequency)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				err := m.checkSites(ctx)
				if err != nil {
					return err
				}
			}
		}
	})

	m.G.Go(func() error {
		ticker := time.NewTicker(PrintTimeout)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				err := m.printStatuses(ctx)
				if err != nil {
					return err
				}
			}
		}
	})

	if err := m.G.Wait(); err != nil {
		return err
	}

	return nil
}

func (m *Monitor) checkSites(ctx context.Context) error {
	client := http.Client{
		Timeout: NetTimeout,
	}

	for _, site := range m.Sites {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, site, nil)
			if err != nil {
				return err
			}

			resp, err := client.Do(req)
			if err != nil {
				return err
			}

			if resp.Body != nil {
				err = resp.Body.Close()
				if err != nil {
					return err
				}
			}

			newSiteStatus := SiteStatus{
				Name:          site,
				StatusCode:    resp.StatusCode,
				TimeOfRequest: time.Now().Truncate(time.Second),
			}

			m.Mtx.Lock()
			m.StatusMap[site] = newSiteStatus
			m.Mtx.Unlock()
		}
	}

	return nil
}

func (m *Monitor) printStatuses(ctx context.Context) error {
	fmt.Printf("------ RESULTS ------\n")
	m.Mtx.Lock()
	for _, siteStatus := range m.StatusMap {
		select {
		case <-ctx.Done():
			fmt.Printf("------\n")
			return ctx.Err()
		default:
			fmt.Printf("%v\t%v\t%v\n", siteStatus.Name, siteStatus.StatusCode, siteStatus.TimeOfRequest.Format(time.RFC3339Nano))
		}
	}
	m.Mtx.Unlock()

	return nil
}
