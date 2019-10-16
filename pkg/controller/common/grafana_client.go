package common

import (
	"bytes"
	"crypto/md5"
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

		resp, err := http.Get(g.api.String() + "/api/dashboards/uid/" + uid)
		if err != nil {
			return false, err
		}
		if resp.StatusCode/100 == 2 {
			return true, nil
		}
		return false, nil
	default:
		return false, errors.New(fmt.Sprintf("unknown kind: %v", kind))
	}
}

func dashboardUID(d *v1alpha1.GrafanaDashboard) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(d.Namespace+d.Name)))
}

func (g *GrafanaClient) UpdateDashboard(d *v1alpha1.GrafanaDashboard, json string) (bool, error) {
	uid := dashboardUID(d)

	jsonb, err := jsonparser.Set([]byte(json), []byte(fmt.Sprintf(`"%s"`, uid)), "uid")
	if err != nil {
		return false, err
	}

	jsonb = []byte(fmt.Sprintf(`{"dashboard": %s}`, string(jsonb)))

	body := bytes.NewBuffer(jsonb)
	req, err := http.NewRequest(http.MethodPost, g.api.String()+"/api/dashboards/db", body)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req) // TODO: Don't use DefaultClient
	if err != nil {
		return false, err
	}

	if resp.StatusCode/100 == 2 {
		log.Info("successfully updated dashboard", d.Namespace, d.Name)
		return true, nil
	}

	log.Info("failed updating dashboard", "status", resp.Status, "namespace", d.Namespace, "name", d.Name)

	respbody, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(respbody))

	return false, nil
}

func (g *GrafanaClient) DeleteDashboard(d *v1alpha1.GrafanaDashboard) error {
	panic("implement me")
}
