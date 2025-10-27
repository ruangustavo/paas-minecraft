package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type CloudflareService struct {
	apiToken   string
	zoneID     string
	serverIP   string
	baseDomain string
	client     *http.Client
}

func NewCloudflareService(apiToken, zoneID, serverIP, baseDomain string) *CloudflareService {
	if baseDomain == "" {
		baseDomain = os.Getenv("BASE_DOMAIN")
	}
	return &CloudflareService{
		apiToken:   apiToken,
		zoneID:     zoneID,
		serverIP:   serverIP,
		baseDomain: baseDomain,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (cs *CloudflareService) BaseDomain() string {
	return cs.baseDomain
}

func (cs *CloudflareService) CreateDNSRecord(subdomain string) error {
	if cs.apiToken == "" || cs.zoneID == "" || cs.serverIP == "" || cs.baseDomain == "" {
		return errors.New("cloudflare service not configured: missing env vars")
	}

	fqdn := subdomain
	if !strings.Contains(subdomain, ".") {
		fqdn = fmt.Sprintf("%s.%s", subdomain, cs.baseDomain)
	}

	payload := map[string]any{
		"type":    "A",
		"name":    fqdn,
		"content": cs.serverIP,
		"ttl":     1,
		"proxied": false,
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", cs.zoneID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cs.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := cs.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var apiResp cloudflareAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("cloudflare create decode: %w", err)
	}
	if !apiResp.Success {
		return fmt.Errorf("cloudflare create failed: %s", apiResp.AllErrors())
	}
	return nil
}

func (cs *CloudflareService) DeleteDNSRecord(subdomain string) error {
	if cs.apiToken == "" || cs.zoneID == "" {
		return errors.New("cloudflare service not configured: missing env vars")
	}

	fqdn := subdomain
	if !strings.Contains(subdomain, ".") {
		fqdn = fmt.Sprintf("%s.%s", subdomain, cs.baseDomain)
	}

	listURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=A&name=%s", cs.zoneID, fqdn)
	listReq, err := http.NewRequest(http.MethodGet, listURL, nil)
	if err != nil {
		return err
	}
	listReq.Header.Set("Authorization", "Bearer "+cs.apiToken)

	listResp, err := cs.client.Do(listReq)
	if err != nil {
		return err
	}
	defer listResp.Body.Close()

	var list cloudflareListResponse
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		return fmt.Errorf("cloudflare list decode: %w", err)
	}
	if !list.Success {
		return fmt.Errorf("cloudflare list failed: %s", list.AllErrors())
	}
	if len(list.Result) == 0 {
		return nil
	}

	id := list.Result[0].ID
	delURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", cs.zoneID, id)
	delReq, err := http.NewRequest(http.MethodDelete, delURL, nil)
	if err != nil {
		return err
	}
	delReq.Header.Set("Authorization", "Bearer "+cs.apiToken)

	delResp, err := cs.client.Do(delReq)
	if err != nil {
		return err
	}
	defer delResp.Body.Close()

	var del cloudflareAPIResponse
	if err := json.NewDecoder(delResp.Body).Decode(&del); err != nil {
		return fmt.Errorf("cloudflare delete decode: %w", err)
	}
	if !del.Success {
		return fmt.Errorf("cloudflare delete failed: %s", del.AllErrors())
	}
	return nil
}

type cloudflareAPIResponse struct {
	Success bool                 `json:"success"`
	Errors  []cloudflareAPIError `json:"errors"`
}

type cloudflareListResponse struct {
	Success bool                    `json:"success"`
	Errors  []cloudflareAPIError    `json:"errors"`
	Result  []cloudflareDNSListItem `json:"result"`
}

type cloudflareDNSListItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type cloudflareAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (r cloudflareAPIResponse) AllErrors() string {
	if len(r.Errors) == 0 {
		return "unknown error"
	}
	var messages []string
	for _, err := range r.Errors {
		messages = append(messages, err.Message)
	}
	return strings.Join(messages, ", ")
}

func (r cloudflareListResponse) AllErrors() string {
	if len(r.Errors) == 0 {
		return "unknown error"
	}
	var messages []string
	for _, err := range r.Errors {
		messages = append(messages, err.Message)
	}
	return strings.Join(messages, ", ")
}
