package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/pennywise/server/cost"
	"github.com/kaytu-io/pennywise/server/internal/ingester"
	"github.com/kaytu-io/pennywise/server/resource"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"strings"
	"time"
)

type EchoError struct {
	Message string `json:"message"`
}

type OnboardServiceClient interface {
	GetResourceCost(req resource.Resource) (*cost.Cost, error)
	GetStateCost(req resource.State) (*cost.Cost, error)
	Ingest(provider, service, region string) error
}

type serverClient struct {
	baseURL string
}

func NewPennywiseServerClient(baseURL string) *serverClient {
	return &serverClient{
		baseURL: baseURL,
	}
}

func (s *serverClient) ListIngestionJobs(provider, service, region, status string) ([]ingester.IngestionJob, error) {
	url := fmt.Sprintf("%s/api/v1/ingest/jobs?status=%s&provider=%s&service=%s&region=%s", s.baseURL, status, provider, service, region)
	url = strings.ReplaceAll(url, " ", "%20")

	var jobs []ingester.IngestionJob
	if statusCode, err := doRequest(http.MethodGet, url, nil, &jobs); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return jobs, nil
}

func (s *serverClient) GetIngestionJob(id string) (*ingester.IngestionJob, error) {
	url := fmt.Sprintf("%s/api/v1/ingest/jobs/%s", s.baseURL, id)

	var job ingester.IngestionJob
	if statusCode, err := doRequest(http.MethodGet, url, nil, &job); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &job, nil
}

func (s *serverClient) Ingest(provider, service, region string) (*ingester.IngestionJob, error) {
	url := fmt.Sprintf("%s/api/v1/ingest?provider=%s&service=%s&region=%s", s.baseURL, provider, service, region)
	url = strings.ReplaceAll(url, " ", "%20")

	var job ingester.IngestionJob
	if statusCode, err := doRequest(http.MethodPut, url, nil, &job); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &job, nil
}

func (s *serverClient) GetResourceCost(req resource.Resource) (*cost.State, error) {
	url := fmt.Sprintf("%s/api/v1/cost/resource", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var cost cost.State
	if statusCode, err := doRequest(http.MethodGet, url, payload, &cost); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &cost, nil
}

func (s *serverClient) GetStateCost(req resource.State) (*cost.State, error) {
	url := fmt.Sprintf("%s/api/v1/cost/state", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var cost cost.State
	if statusCode, err := doRequest(http.MethodGet, url, payload, &cost); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &cost, nil
}

func doRequest(method, url string, payload []byte, v interface{}) (statusCode int, err error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return statusCode, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set(echo.HeaderContentType, "application/json")
	t := http.DefaultTransport.(*http.Transport)
	client := http.Client{
		Timeout:   3 * time.Minute,
		Transport: t,
	}

	res, err := client.Do(req)
	if err != nil {
		return statusCode, fmt.Errorf("do request: %w", err)
	}
	defer res.Body.Close()
	body := res.Body

	statusCode = res.StatusCode
	if res.StatusCode != http.StatusOK {
		d, err := io.ReadAll(body)
		if err != nil {
			return statusCode, fmt.Errorf("read body: %w", err)
		}

		var echoerr EchoError
		if jserr := json.Unmarshal(d, &echoerr); jserr == nil {
			return statusCode, fmt.Errorf(echoerr.Message)
		}

		return statusCode, fmt.Errorf("http status: %d: %s", res.StatusCode, d)
	}
	if v == nil {
		return statusCode, nil
	}

	return statusCode, json.NewDecoder(body).Decode(v)
}
