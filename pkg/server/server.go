package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/pennywise/pkg/cost"
	"github.com/kaytu-io/pennywise/pkg/schema"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"strings"
	"time"
)

type EchoError struct {
	Message string `json:"message"`
}

type ServerClient interface {
	GetStateCost(req schema.Submission) (*cost.State, error)
	GetStateCostV2(req schema.SubmissionV2) (*cost.ModularState, error)
	AddIngestion(provider, service, region string) (*schema.IngestionJob, error)
	ListIngestionJobs(provider, service, region, status string) ([]schema.IngestionJob, error)
	GetIngestionJob(id string) (*schema.IngestionJob, error)
	ListServices(provider string) ([]string, error)
	GetSubmissionsDiff(req schema.SubmissionsDiff) (*schema.StateDiff, error)
	GetSubmissionsDiffV2(req schema.SubmissionsDiffV2) (*schema.ModularStateDiff, error)
}

type serverClient struct {
	baseURL string
	config  *Config
}

func NewPennywiseServerClient(baseURL string) (ServerClient, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	}
	return &serverClient{baseURL: baseURL, config: config}, nil
}

func (s *serverClient) ListServices(provider string) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/ingestion/new_services?provider=%s", s.baseURL, provider)
	url = strings.ReplaceAll(url, " ", "%20")

	var listNewServices []string
	if statusCode, err := s.doRequest(http.MethodGet, url, nil, &listNewServices); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return listNewServices, nil
}

func (s *serverClient) ListIngestionJobs(provider, service, region, status string) ([]schema.IngestionJob, error) {
	url := fmt.Sprintf("%s/api/v1/ingestion/jobs?status=%s&provider=%s&service=%s&region=%s", s.baseURL, status, provider, service, region)
	url = strings.ReplaceAll(url, " ", "%20")

	var jobs []schema.IngestionJob
	if statusCode, err := s.doRequest(http.MethodGet, url, nil, &jobs); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		if strings.Contains(err.Error(), "connect: connection refused") {
			return nil, fmt.Errorf("Can't connect to the server. Please ensure that your server is running or that you have entered the --server-url flag currectly ")
		}
		return nil, err
	}
	return jobs, nil
}

func (s *serverClient) GetIngestionJob(id string) (*schema.IngestionJob, error) {
	url := fmt.Sprintf("%s/api/v1/ingestion/jobs/%s", s.baseURL, id)

	var job schema.IngestionJob
	if statusCode, err := s.doRequest(http.MethodGet, url, nil, &job); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			if strings.Contains(err.Error(), "this ID does not exist") {
				return nil, fmt.Errorf("this ID does not exist")
			}
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		if strings.Contains(err.Error(), "connect: connection refused") {
			return nil, fmt.Errorf("Can't connect to the server. Please ensure that your server is running or that you have entered the --server-url flag currectly ")
		}
		return nil, err
	}
	return &job, nil
}

func (s *serverClient) AddIngestion(provider, service, region string) (*schema.IngestionJob, error) {
	url := fmt.Sprintf("%s/api/v1/ingestion/jobs?provider=%s&service=%s&region=%s", s.baseURL, provider, service, region)
	url = strings.ReplaceAll(url, " ", "%20")

	var job schema.IngestionJob
	if statusCode, err := s.doRequest(http.MethodPut, url, nil, &job); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			if strings.Contains(err.Error(), "this service is not supported") {
				return nil, fmt.Errorf("this services is not supported")
			}
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		if strings.Contains(err.Error(), "connect: connection refused") {
			return nil, fmt.Errorf("Can't connect to the server. Please ensure that your server is running or that you have entered the --server-url flag currectly ")
		}
		return nil, err
	}
	return &job, nil
}

func (s *serverClient) GetStateCost(req schema.Submission) (*cost.State, error) {
	url := fmt.Sprintf("%s/api/v1/cost/submission", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var cost cost.State
	if statusCode, err := s.doRequest(http.MethodGet, url, payload, &cost); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		if strings.Contains(err.Error(), "connect: connection refused") {
			return nil, fmt.Errorf("Can't connect to the server. Please ensure that your server is running or that you have entered the --server-url flag currectly ")
		}
		return nil, err
	}
	return &cost, nil
}

func (s *serverClient) GetStateCostV2(req schema.SubmissionV2) (*cost.ModularState, error) {
	url := fmt.Sprintf("%s/api/v2/cost/submission", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var cost cost.ModularState
	if statusCode, err := s.doRequest(http.MethodGet, url, payload, &cost); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		if strings.Contains(err.Error(), "connect: connection refused") {
			return nil, fmt.Errorf("Can't connect to the server. Please ensure that your server is running or that you have entered the --server-url flag currectly ")
		}
		return nil, err
	}
	return &cost, nil
}

func (s *serverClient) GetSubmissionsDiff(req schema.SubmissionsDiff) (*schema.StateDiff, error) {
	url := fmt.Sprintf("%s/api/v1/cost/diff", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var cost schema.StateDiff
	if statusCode, err := s.doRequest(http.MethodGet, url, payload, &cost); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		if strings.Contains(err.Error(), "connect: connection refused") {
			return nil, fmt.Errorf("Can't connect to the server. Please ensure that your server is running or that you have entered the --server-url flag currectly ")
		}
		return nil, err
	}
	return &cost, nil
}

func (s *serverClient) GetSubmissionsDiffV2(req schema.SubmissionsDiffV2) (*schema.ModularStateDiff, error) {
	url := fmt.Sprintf("%s/api/v2/cost/diff", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var cost schema.ModularStateDiff
	if statusCode, err := s.doRequest(http.MethodGet, url, payload, &cost); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		if strings.Contains(err.Error(), "connect: connection refused") {
			return nil, fmt.Errorf("Can't connect to the server. Please ensure that your server is running or that you have entered the --server-url flag currectly ")
		}
		return nil, err
	}
	return &cost, nil
}

func (s *serverClient) doRequest(method, url string, payload []byte, v interface{}) (statusCode int, err error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return statusCode, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set(echo.HeaderContentType, "application/json")
	req.Header.Set(strings.ToLower(echo.HeaderAuthorization), "Bearer "+s.config.AccessToken)
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
