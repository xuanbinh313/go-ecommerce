package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/payment/domain"
)

const baseURL = "https://api.stripe.com/v1"

type Options struct {
	APIKey                     string
	WebhookSecret              string
	Capture                    bool
	AutomaticPaymentMethods    bool
	PaymentMethodConfiguration string
	PaymentDescription         string
	OxxoExpiresDays            int
	AsyncPaymentMethodTypes    []string
}

type Provider struct {
	options Options
	variant string
	client  *http.Client
}

func New(options Options, variant string) (*Provider, error) {
	if options.APIKey == "" {
		return nil, errors.New("Required option `apiKey` is missing in Stripe plugin")
	}
	if options.OxxoExpiresDays <= 0 {
		options.OxxoExpiresDays = 3
	}
	if variant == "" {
		variant = "stripe"
	}
	return &Provider{options: options, variant: variant, client: &http.Client{Timeout: 25 * time.Second}}, nil
}

func (p *Provider) CreateSession(ctx context.Context, input map[string]any) (map[string]any, error) {
	data := mapValue(input["data"])
	contextData := mapValue(input["context"])
	currency := stringValue(input["currency_code"])
	amount, err := smallestUnit(input["amount"], currency)
	if err != nil {
		return nil, err
	}
	form := url.Values{"amount": {strconv.FormatInt(amount, 10)}, "currency": {strings.ToLower(currency)}}
	metadata := mapValue(data["metadata"])
	if sessionID := stringValue(data["session_id"]); sessionID != "" {
		metadata["session_id"] = sessionID
	}
	flatten(form, "metadata", metadata)
	params := p.paymentIntentParams(data)
	for key, value := range params {
		flattenValue(form, key, value)
	}
	if account := mapValue(contextData["account_holder"]); len(account) > 0 {
		if id := stringValue(mapValue(account["data"])["id"]); id != "" {
			form.Set("customer", id)
		}
	}
	result, err := p.request(ctx, http.MethodPost, "/payment_intents", form, stringValue(contextData["idempotency_key"]))
	if err != nil {
		var stripeErr *apiError
		if errors.As(err, &stripeErr) && stripeErr.PaymentIntent != nil {
			intent := stripeErr.PaymentIntent
			return map[string]any{"id": fallback(stringValue(data["session_id"]), stringValue(intent["id"])), "data": intent, "status": p.status(intent)}, nil
		}
		if errors.As(err, &stripeErr) && stripeErr.Type == "api_error" {
			return map[string]any{"id": stringValue(data["session_id"]), "data": map[string]any{"indeterminate_due_to": "stripe_api_error"}, "status": "pending"}, nil
		}
		return nil, err
	}
	return map[string]any{"id": result["id"], "data": result, "status": p.status(result)}, nil
}

func (p *Provider) UpdateSession(ctx context.Context, input map[string]any) (map[string]any, error) {
	data := mapValue(input["data"])
	id := stringValue(data["id"])
	if id == "" {
		return nil, errors.New("An error occurred in updatePayment: missing payment intent ID")
	}
	amount, err := smallestUnit(input["amount"], stringValue(input["currency_code"]))
	if err != nil {
		return nil, err
	}
	if current, ok := numberInt(data["amount"]); ok && current == amount {
		return map[string]any{"data": data, "status": p.status(data)}, nil
	}
	form := url.Values{"amount": {strconv.FormatInt(amount, 10)}, "expand[]": {"payment_method"}}
	result, err := p.request(ctx, http.MethodPost, "/payment_intents/"+url.PathEscape(id), form, stringValue(mapValue(input["context"])["idempotency_key"]))
	if err != nil {
		return nil, fmt.Errorf("An error occurred in updatePayment: %w", err)
	}
	return map[string]any{"data": result, "status": p.status(result)}, nil
}

func (p *Provider) DeleteSession(ctx context.Context, input map[string]any) (map[string]any, error) {
	return p.CancelPayment(ctx, input)
}
func (p *Provider) AuthorizePayment(ctx context.Context, input map[string]any) (map[string]any, error) {
	data := mapValue(input["data"])
	id := stringValue(data["id"])
	if id == "" {
		return nil, errors.New("No payment intent ID provided while getting payment status")
	}
	intent, err := p.request(ctx, http.MethodGet, "/payment_intents/"+url.PathEscape(id)+"?expand[]=payment_method", nil, "")
	if err != nil {
		return nil, err
	}
	return map[string]any{"data": intent, "status": p.status(intent)}, nil
}
func (p *Provider) GetStatus(ctx context.Context, input map[string]any) (string, error) {
	result, err := p.AuthorizePayment(ctx, input)
	if err != nil {
		return "", err
	}
	return stringValue(result["status"]), nil
}
func (p *Provider) CapturePayment(ctx context.Context, input map[string]any) (map[string]any, error) {
	data := mapValue(input["data"])
	id := stringValue(data["id"])
	if id == "" {
		return nil, errors.New("An error occurred in capturePayment: missing payment intent ID")
	}
	res, err := p.request(ctx, http.MethodPost, "/payment_intents/"+url.PathEscape(id)+"/capture", url.Values{}, stringValue(mapValue(input["context"])["idempotency_key"]))
	if err != nil {
		var stripeErr *apiError
		if errors.As(err, &stripeErr) && stripeErr.Code == "payment_intent_unexpected_state" && stringValue(mapValue(stripeErr.PaymentIntent)["status"]) == "succeeded" {
			return map[string]any{"data": stripeErr.PaymentIntent}, nil
		}
		return nil, fmt.Errorf("An error occurred in capturePayment: %w", err)
	}
	return map[string]any{"data": res}, nil
}
func (p *Provider) CancelPayment(ctx context.Context, input map[string]any) (map[string]any, error) {
	data := mapValue(input["data"])
	id := stringValue(data["id"])
	if id == "" {
		return map[string]any{"data": data}, nil
	}
	res, err := p.request(ctx, http.MethodPost, "/payment_intents/"+url.PathEscape(id)+"/cancel", url.Values{}, stringValue(mapValue(input["context"])["idempotency_key"]))
	if err != nil {
		var stripeErr *apiError
		if errors.As(err, &stripeErr) && stripeErr.Code == "payment_intent_unexpected_state" && stringValue(mapValue(stripeErr.PaymentIntent)["status"]) == "canceled" {
			return map[string]any{"data": stripeErr.PaymentIntent}, nil
		}
		return nil, fmt.Errorf("An error occurred in cancelPayment: %w", err)
	}
	return map[string]any{"data": res}, nil
}
func (p *Provider) RefundPayment(ctx context.Context, input map[string]any) (map[string]any, error) {
	data := mapValue(input["data"])
	id := stringValue(data["id"])
	if id == "" {
		return nil, errors.New("No payment intent ID provided while refunding payment")
	}
	amount, err := smallestUnit(input["amount"], stringValue(data["currency"]))
	if err != nil {
		return nil, err
	}
	form := url.Values{"amount": {strconv.FormatInt(amount, 10)}, "payment_intent": {id}}
	if _, err = p.request(ctx, http.MethodPost, "/refunds", form, stringValue(mapValue(input["context"])["idempotency_key"])); err != nil && !strings.Contains(err.Error(), "charge_already_refunded") {
		return nil, fmt.Errorf("An error occurred in refundPayment: %w", err)
	}
	return map[string]any{"data": data}, nil
}

func (p *Provider) GetWebhookActionAndData(ctx context.Context, input map[string]any) (map[string]any, error) {
	payload := mapValue(input["payload"])
	headers := mapValue(payload["headers"])
	raw := payload["rawData"]
	if raw == nil {
		raw = payload["raw_data"]
	}
	body, err := rawBytes(raw)
	if err != nil {
		return nil, err
	}
	event, err := p.verifyWebhook(body, stringValue(headers["stripe-signature"]))
	if err != nil {
		return nil, fmt.Errorf("An error occurred in constructWebhookEvent: %w", err)
	}
	eventType := stringValue(event["type"])
	eventData := mapValue(mapValue(event["data"])["object"])
	metadata := mapValue(eventData["metadata"])
	sessionID := stringValue(metadata["session_id"])
	if sessionID == "" {
		return map[string]any{"action": "not_supported"}, nil
	}
	currency := stringValue(eventData["currency"])
	amount := eventData["amount"]
	action := "not_supported"
	switch eventType {
	case "payment_intent.created", "payment_intent.processing":
		method := mapValue(eventData["payment_method"])
		if methodID, ok := eventData["payment_method"].(string); ok {
			method, err = p.request(ctx, http.MethodGet, "/payment_methods/"+url.PathEscape(methodID), nil, "")
			if err != nil {
				return nil, err
			}
		}
		if contains(p.options.AsyncPaymentMethodTypes, stringValue(method["type"])) {
			action = "pending_authorization"
		} else {
			action = "pending"
		}
	case "payment_intent.canceled":
		action = "canceled"
	case "payment_intent.payment_failed":
		action = "failed"
	case "payment_intent.requires_action":
		action = "requires_more"
	case "payment_intent.amount_capturable_updated":
		action = "authorized"
		amount = eventData["amount_capturable"]
	case "payment_intent.partially_funded":
		action = "requires_more"
		instructions := mapValue(mapValue(eventData["next_action"])["display_bank_transfer_instructions"])
		if instructions["amount_remaining"] != nil {
			amount = instructions["amount_remaining"]
		}
	case "payment_intent.succeeded":
		action = "successful"
		amount = eventData["amount_received"]
	}
	converted, err := amountFromSmallest(amount, currency)
	if err != nil {
		return nil, err
	}
	return map[string]any{"action": action, "data": map[string]any{"session_id": sessionID, "amount": converted}}, nil
}

func (p *Provider) CreateAccountHolder(ctx context.Context, input map[string]any) (map[string]any, error) {
	ctxData := mapValue(input["context"])
	holder := mapValue(ctxData["account_holder"])
	if id := stringValue(mapValue(holder["data"])["id"]); id != "" {
		return map[string]any{"id": id}, nil
	}
	customer := mapValue(ctxData["customer"])
	if len(customer) == 0 {
		return nil, errors.New("No customer in context")
	}
	form := url.Values{}
	setIf(form, "email", customer["email"])
	name := stringValue(customer["company_name"])
	if name == "" {
		name = strings.TrimSpace(stringValue(customer["first_name"]) + " " + stringValue(customer["last_name"]))
	}
	setIf(form, "name", name)
	setIf(form, "phone", customer["phone"])
	flattenAddress(form, "address", mapValue(mapValue(customer["billing_address"])))
	res, err := p.request(ctx, http.MethodPost, "/customers", form, stringValue(ctxData["idempotency_key"]))
	if err != nil {
		return nil, fmt.Errorf("An error occurred in createAccountHolder when creating a Stripe customer: %w", err)
	}
	return map[string]any{"id": res["id"], "data": res}, nil
}
func (p *Provider) UpdateAccountHolder(ctx context.Context, input map[string]any) (map[string]any, error) {
	ctxData := mapValue(input["context"])
	holder := mapValue(ctxData["account_holder"])
	id := stringValue(mapValue(holder["data"])["id"])
	if id == "" {
		return nil, errors.New("No account holder in context")
	}
	customer := mapValue(ctxData["customer"])
	if len(customer) == 0 {
		return map[string]any{}, nil
	}
	form := customerForm(customer)
	res, err := p.request(ctx, http.MethodPost, "/customers/"+url.PathEscape(id), form, stringValue(ctxData["idempotency_key"]))
	if err != nil {
		return nil, fmt.Errorf("An error occurred in updateAccountHolder when updating a Stripe customer: %w", err)
	}
	return map[string]any{"data": res}, nil
}
func (p *Provider) DeleteAccountHolder(ctx context.Context, input map[string]any) (map[string]any, error) {
	holder := mapValue(mapValue(input["context"])["account_holder"])
	id := stringValue(mapValue(holder["data"])["id"])
	if id == "" {
		return nil, errors.New("No account holder in context")
	}
	if _, err := p.request(ctx, http.MethodDelete, "/customers/"+url.PathEscape(id), nil, ""); err != nil {
		return nil, fmt.Errorf("An error occurred in deleteAccountHolder: %w", err)
	}
	return map[string]any{}, nil
}
func (p *Provider) ListPaymentMethods(ctx context.Context, input map[string]any) ([]map[string]any, error) {
	ctxData := mapValue(input["context"])
	holder := mapValue(ctxData["account_holder"])
	id := stringValue(mapValue(holder["data"])["id"])
	if id == "" {
		return []map[string]any{}, nil
	}
	res, err := p.request(ctx, http.MethodGet, "/customers/"+url.PathEscape(id)+"/payment_methods?limit=100", nil, "")
	if err != nil {
		return nil, err
	}
	list := mapValue(res)
	items, _ := list["data"].([]any)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		m := mapValue(item)
		out = append(out, map[string]any{"id": m["id"], "data": m})
	}
	return out, nil
}
func (p *Provider) SavePaymentMethod(ctx context.Context, input map[string]any) (map[string]any, error) {
	ctxData := mapValue(input["context"])
	holder := mapValue(ctxData["account_holder"])
	customerID := stringValue(mapValue(holder["data"])["id"])
	if customerID == "" {
		return nil, errors.New("Account holder not set while saving a payment method")
	}
	form := url.Values{"customer": {customerID}}
	flatten(form, "", mapValue(input["data"]))
	res, err := p.request(ctx, http.MethodPost, "/setup_intents", form, stringValue(ctxData["idempotency_key"]))
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": res["id"], "data": res}, nil
}
func (p *Provider) DeletePaymentMethod(ctx context.Context, input map[string]any) (map[string]any, error) {
	id := stringValue(mapValue(input["data"])["id"])
	if id == "" {
		return nil, errors.New("Payment method ID not set while deleting a payment method")
	}
	if _, err := p.request(ctx, http.MethodPost, "/payment_methods/"+url.PathEscape(id)+"/detach", url.Values{}, stringValue(mapValue(input["context"])["idempotency_key"])); err != nil {
		return nil, err
	}
	return map[string]any{}, nil
}

func (p *Provider) paymentIntentParams(data map[string]any) map[string]any {
	params := map[string]any{}
	if desc := data["payment_description"]; desc != nil {
		params["description"] = desc
	} else if p.options.PaymentDescription != "" {
		params["description"] = p.options.PaymentDescription
	}
	capture := "manual"
	if p.options.Capture {
		capture = "automatic"
	}
	if value := data["capture_method"]; value != nil {
		capture = stringValue(value)
	}
	params["capture_method"] = capture
	if data["setup_future_usage"] != nil {
		params["setup_future_usage"] = data["setup_future_usage"]
	}
	variantMethods := map[string]string{"stripe-oxxo": "oxxo", "stripe-bancontact": "bancontact", "stripe-blik": "blik", "stripe-giropay": "giropay", "stripe-ideal": "ideal", "stripe-przelewy24": "p24", "stripe-promptpay": "promptpay"}
	if method, ok := variantMethods[p.variant]; ok {
		params["payment_method_types"] = []any{method}
		params["capture_method"] = "automatic"
		if method == "oxxo" {
			params["payment_method_options"] = map[string]any{"oxxo": map[string]any{"expires_after_days": p.options.OxxoExpiresDays}}
		}
	}
	if params["payment_method_types"] == nil && p.options.PaymentMethodConfiguration != "" {
		params["payment_method_configuration"] = p.options.PaymentMethodConfiguration
	}
	if params["automatic_payment_methods"] == nil && p.options.AutomaticPaymentMethods {
		params["automatic_payment_methods"] = map[string]any{"enabled": true}
	}
	for _, key := range []string{"payment_method_types", "payment_method_data", "payment_method_options", "automatic_payment_methods", "off_session", "confirm", "payment_method", "return_url", "shared_payment_token", "payment_method_configuration", "setup_future_usage", "capture_method", "description"} {
		if value, ok := data[key]; ok {
			params[key] = value
		}
	}
	return params
}

func (p *Provider) status(intent map[string]any) string {
	switch stringValue(intent["status"]) {
	case "requires_payment_method":
		if intent["last_payment_error"] != nil {
			return "error"
		}
		return "pending"
	case "requires_confirmation":
		return "pending"
	case "processing":
		method := mapValue(intent["payment_method"])
		if contains(p.options.AsyncPaymentMethodTypes, stringValue(method["type"])) {
			return "pending_authorization"
		}
		return "pending"
	case "requires_action":
		return "requires_more"
	case "canceled":
		return "canceled"
	case "requires_capture":
		return "authorized"
	case "succeeded":
		return "captured"
	default:
		return "pending"
	}
}

func (p *Provider) request(ctx context.Context, method, path string, form url.Values, idempotencyKey string) (map[string]any, error) {
	var last error
	for attempt := 0; attempt <= 3; attempt++ {
		var body io.Reader
		if form != nil {
			body = strings.NewReader(form.Encode())
		}
		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(p.options.APIKey, "")
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
		resp, err := p.client.Do(req)
		if err != nil {
			last = err
			if attempt < 3 {
				if waitErr := waitContext(ctx, time.Duration(1<<attempt)*time.Second); waitErr == nil {
					continue
				} else {
					return nil, waitErr
				}
			}
			break
		}
		payload, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			result := map[string]any{}
			if len(payload) > 0 {
				if err = json.Unmarshal(payload, &result); err != nil {
					return nil, err
				}
			}
			return result, nil
		}
		var apiErr struct {
			Error struct {
				Type          string         `json:"type"`
				Code          string         `json:"code"`
				Message       string         `json:"message"`
				PaymentIntent map[string]any `json:"payment_intent"`
			} `json:"error"`
		}
		_ = json.Unmarshal(payload, &apiErr)
		last = &apiError{Status: resp.StatusCode, Type: apiErr.Error.Type, Code: apiErr.Error.Code, Message: apiErr.Error.Message, PaymentIntent: apiErr.Error.PaymentIntent}
		if resp.StatusCode == 429 && attempt < 3 {
			if err = waitContext(ctx, time.Duration(1<<attempt)*time.Second); err == nil {
				continue
			}
		}
		return nil, last
	}
	return nil, last
}

func (p *Provider) verifyWebhook(raw []byte, signature string) (map[string]any, error) {
	if p.options.WebhookSecret == "" {
		return nil, errors.New("webhook secret is required")
	}
	var timestamp string
	var sigs []string
	for _, part := range strings.Split(signature, ",") {
		pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(pair) != 2 {
			continue
		}
		if pair[0] == "t" {
			timestamp = pair[1]
		} else if pair[0] == "v1" {
			sigs = append(sigs, pair[1])
		}
	}
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || math.Abs(float64(time.Now().Unix()-seconds)) > 300 {
		return nil, errors.New("webhook timestamp outside tolerance")
	}
	mac := hmac.New(sha256.New, []byte(p.options.WebhookSecret))
	_, _ = mac.Write([]byte(timestamp + "."))
	_, _ = mac.Write(raw)
	expected := mac.Sum(nil)
	valid := false
	for _, sig := range sigs {
		decoded, e := hex.DecodeString(sig)
		if e == nil && hmac.Equal(decoded, expected) {
			valid = true
			break
		}
	}
	if !valid {
		return nil, errors.New("webhook signature verification failed")
	}
	event := map[string]any{}
	err = json.Unmarshal(raw, &event)
	return event, err
}

type apiError struct {
	Status              int
	Type, Code, Message string
	PaymentIntent       map[string]any
}

func (e *apiError) Error() string { return e.Code + ": " + e.Message }
func smallestUnit(value any, currency string) (int64, error) {
	amount, err := decimalNumber(value)
	if err != nil {
		return 0, err
	}
	power := 2
	zero := strings.Fields("BIF CLP DJF GNF JPY KMF KRW MGA PYG RWF UGX VND VUV XAF XOF XPF")
	three := strings.Fields("BHD IQD JOD KWD OMR TND")
	for _, code := range zero {
		if strings.EqualFold(code, currency) {
			power = 0
		}
	}
	for _, code := range three {
		if strings.EqualFold(code, currency) {
			power = 3
		}
	}
	multiplier := math.Pow10(power)
	rounded := math.Round(amount*multiplier) / multiplier
	smallest := math.Round(rounded * multiplier)
	if power == 3 {
		smallest = math.Ceil(smallest/10) * 10
	}
	return int64(smallest), nil
}
func amountFromSmallest(value any, currency string) (float64, error) {
	n, err := decimalNumber(value)
	if err != nil {
		return 0, err
	}
	_, pow := smallestUnitPower(currency)
	return n / math.Pow10(pow), nil
}
func smallestUnitPower(currency string) (int, int) {
	_, _ = currency, 0
	power := 2
	for _, c := range strings.Fields("BIF CLP DJF GNF JPY KMF KRW MGA PYG RWF UGX VND VUV XAF XOF XPF") {
		if strings.EqualFold(c, currency) {
			power = 0
		}
	}
	for _, c := range strings.Fields("BHD IQD JOD KWD OMR TND") {
		if strings.EqualFold(c, currency) {
			power = 3
		}
	}
	return 0, power
}
func decimalNumber(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case json.Number:
		return n.Float64()
	case string:
		return strconv.ParseFloat(n, 64)
	default:
		return 0, fmt.Errorf("invalid monetary amount %v", v)
	}
}
func numberInt(v any) (int64, bool) { n, err := decimalNumber(v); return int64(n), err == nil }
func mapValue(v any) map[string]any {
	if m, ok := v.(map[string]any); ok && m != nil {
		return m
	}
	return map[string]any{}
}
func stringValue(v any) string { s, _ := v.(string); return s }
func fallback(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
func rawBytes(v any) ([]byte, error) {
	switch x := v.(type) {
	case []byte:
		return x, nil
	case string:
		return []byte(x), nil
	case json.RawMessage:
		return x, nil
	default:
		return nil, fmt.Errorf("expected raw webhook bytes, got %T", v)
	}
}
func waitContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
func flatten(form url.Values, prefix string, m map[string]any) {
	for key, value := range m {
		name := key
		if prefix != "" {
			name = prefix + "[" + key + "]"
		}
		flattenValue(form, name, value)
	}
}
func flattenValue(form url.Values, key string, v any) {
	switch value := v.(type) {
	case map[string]any:
		flatten(form, key, value)
	case []string:
		for i, item := range value {
			form.Set(fmt.Sprintf("%s[%d]", key, i), item)
		}
	case []any:
		for i, item := range value {
			flattenValue(form, fmt.Sprintf("%s[%d]", key, i), item)
		}
	case nil:
	case bool:
		form.Set(key, strconv.FormatBool(value))
	default:
		form.Set(key, fmt.Sprint(value))
	}
}
func setIf(form url.Values, key string, v any) {
	if v != nil && fmt.Sprint(v) != "" {
		flattenValue(form, key, v)
	}
}
func customerForm(customer map[string]any) url.Values {
	form := url.Values{}
	setIf(form, "email", customer["email"])
	name := stringValue(customer["company_name"])
	if name == "" {
		name = strings.TrimSpace(stringValue(customer["first_name"]) + " " + stringValue(customer["last_name"]))
	}
	setIf(form, "name", name)
	setIf(form, "phone", customer["phone"])
	flattenAddress(form, "address", mapValue(customer["billing_address"]))
	return form
}
func flattenAddress(form url.Values, prefix string, address map[string]any) {
	mapping := map[string]string{"city": "city", "country_code": "country", "address_1": "line1", "address_2": "line2", "postal_code": "postal_code", "province": "state"}
	for src, dest := range mapping {
		if value := address[src]; value != nil {
			form.Set(prefix+"["+dest+"]", fmt.Sprint(value))
		}
	}
}

var _ domain.Provider = (*Provider)(nil)
