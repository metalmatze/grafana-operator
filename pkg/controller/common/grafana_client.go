package common

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/buger/jsonparser"
	"github.com/integr8ly/grafana-operator/pkg/apis/integreatly/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

type GrafanaClient struct {
	api *url.URL
}

func NewGrafanaClient() *GrafanaClient {
	return &GrafanaClient{
		api: &url.URL{Scheme: "http", Host: "localhost:3000"},
	}
}

func (g *GrafanaClient) IsKnown(kind string, o runtime.Object) (bool, error) {
	switch kind {
	case v1alpha1.GrafanaDashboardKind:
		d := o.(*v1alpha1.GrafanaDashboard)
		uid := dashboardUID(d)

		req, err := http.NewRequest(http.MethodGet, g.api.String()+"/api/dashboards/uid/"+uid, nil)
		if err != nil {
			return false, err
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "Bearer eyJrIjoiQ0VLSTZ0TXR6ZjlBNllUMm5FOGc2TWgySXUzVnVZYk0iLCJuIjoib3BlcmF0b3IiLCJpZCI6MX0=")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, err
		}
		if resp.StatusCode/100 == 2 {
			return true, nil
		}

		// TODO: Compare versions of dashboard

		return false, nil
	default:
		return false, errors.New(fmt.Sprintf("unknown kind: %v", kind))
	}
}

func dashboardUID(d *v1alpha1.GrafanaDashboard) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(d.Namespace+d.Name)))
}

func (g *GrafanaClient) UpdateDashboard(d *v1alpha1.GrafanaDashboard, j string) (bool, error) {
	uid := dashboardUID(d)

	jsonb, err := jsonparser.Set([]byte(j), []byte(fmt.Sprintf(`"%s"`, uid)), "uid")
	if err != nil {
		return false, err
	}

	body := []byte(fmt.Sprintf(`{"dashboard": %s, "overwrite": true}`, string(jsonb)))

	req, err := http.NewRequest(http.MethodPost, g.api.String()+"/api/dashboards/db", bytes.NewBuffer(body))
	if err != nil {
		return false, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer eyJrIjoiQ0VLSTZ0TXR6ZjlBNllUMm5FOGc2TWgySXUzVnVZYk0iLCJuIjoib3BlcmF0b3IiLCJpZCI6MX0=")

	resp, err := http.DefaultClient.Do(req) // TODO: Don't use DefaultClient
	if err != nil {
		return false, err
	}

	if resp.StatusCode/100 == 2 {
		log.Info("successfully updated dashboard", d.Namespace, d.Name)
		return true, nil
	}

	defer resp.Body.Close()

	var respPayload struct {
		Message string `json:"message"`
	}

	err = json.NewDecoder(resp.Body).Decode(&respPayload)
	if err != nil {
		// TODO
	}

	log.Info("failed updating dashboard",
		"status", resp.Status,
		"namespace", d.Namespace,
		"name", d.Name,
		"response", respPayload.Message,
	)

	return false, errors.New(respPayload.Message)
}

func (g *GrafanaClient) DeleteDashboard(d *v1alpha1.GrafanaDashboard) error {
	uid := dashboardUID(d)

	req, err := http.NewRequest(http.MethodDelete, g.api.String()+"/api/dashboards/uid/"+uid, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer eyJrIjoiQ0VLSTZ0TXR6ZjlBNllUMm5FOGc2TWgySXUzVnVZYk0iLCJuIjoib3BlcmF0b3IiLCJpZCI6MX0=")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode/100 == 2 {
		log.Info("successfully deleted dashboard", d.Namespace, d.Name)
		return nil
	}

	defer resp.Body.Close()
	respbody, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(respbody))

	return errors.New(fmt.Sprintf("unexpected response: %s", resp.Status))
}
