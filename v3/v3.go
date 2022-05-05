package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/superoo7/go-gecko/format"
	"github.com/superoo7/go-gecko/v3/types"
)

var baseURL = "https://api.coingecko.com/api/v3"

// Client struct
type Client struct {
	httpClient *http.Client
	StartHook  func()
	EndHook    func()
}

type option func(c *Client)

func WithStartHook(hook func()) func(c *Client) {
	return func(c *Client) {
		c.StartHook = hook
	}
}

func WithEndHook(hook func()) func(c *Client) {
	return func(c *Client) {
		c.EndHook = hook
	}
}

func WithHTTPClient(cl *http.Client) func(c *Client) {
	return func(c *Client) {
		c.httpClient = cl
	}
}

// NewClient create new client object
func NewClient(opts ...option) *Client {
	cl := &Client{}
	for _, o := range opts {
		o(cl)
	}

	if cl.httpClient == nil {
		cl.httpClient = http.DefaultClient
	}

	return cl
}

// helper
// doReq HTTP client
func doReq(req *http.Request, client *http.Client) ([]byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if 200 != resp.StatusCode {
		return nil, fmt.Errorf("%s", body)
	}
	return body, nil
}

// doStreamingReq HTTP client
func doStreamingReq(req *http.Request, client *http.Client) (io.ReadCloser, error) {
	resp, err := client.Do(req)
	if err != nil {
		return resp.Body, err
	}

	if 200 != resp.StatusCode {
		return resp.Body, fmt.Errorf("return status code is %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// MakeReq HTTP request helper
func (c *Client) MakeReq(url string) ([]byte, error) {
	if c.StartHook != nil {
		c.StartHook()
	}
	if c.EndHook != nil {
		defer c.EndHook()
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := doReq(req, c.httpClient)
	if err != nil {
		return nil, err
	}

	return resp, err
}

// MakeStreamingReq HTTP request helper
func (c *Client) MakeStreamingReq(ctx context.Context, url string) (io.ReadCloser, error) {
	if c.StartHook != nil {
		c.StartHook()
	}
	if c.EndHook != nil {
		defer c.EndHook()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := doStreamingReq(req, c.httpClient)
	if err != nil {
		if resp != nil {
			defer resp.Close()
			io.Copy(io.Discard, resp)
		}

		return nil, err
	}

	return resp, err
}

// API

// Ping /ping endpoint
func (c *Client) Ping() (*types.Ping, error) {
	url := fmt.Sprintf("%s/ping", baseURL)
	resp, err := c.MakeReq(url)
	if err != nil {
		return nil, err
	}
	var data *types.Ping
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// SimpleSinglePrice /simple/price  Single ID and Currency (ids, vs_currency)
func (c *Client) SimpleSinglePrice(id string, vsCurrency string) (*types.SimpleSinglePrice, error) {
	idParam := []string{strings.ToLower(id)}
	vcParam := []string{strings.ToLower(vsCurrency)}

	t, err := c.SimplePrice(idParam, vcParam)
	if err != nil {
		return nil, err
	}
	curr := (*t)[id]
	if len(curr) == 0 {
		return nil, fmt.Errorf("id or vsCurrency not existed")
	}
	data := &types.SimpleSinglePrice{ID: id, Currency: vsCurrency, MarketPrice: curr[vsCurrency]}
	return data, nil
}

// SimplePrice /simple/price Multiple ID and Currency (ids, vs_currencies)
func (c *Client) SimplePrice(ids []string, vsCurrencies []string) (*map[string]map[string]float32, error) {
	params := url.Values{}
	idsParam := strings.Join(ids[:], ",")
	vsCurrenciesParam := strings.Join(vsCurrencies[:], ",")

	params.Add("ids", idsParam)
	params.Add("vs_currencies", vsCurrenciesParam)

	url := fmt.Sprintf("%s/simple/price?%s", baseURL, params.Encode())
	resp, err := c.MakeReq(url)
	if err != nil {
		return nil, err
	}

	t := make(map[string]map[string]float32)
	err = json.Unmarshal(resp, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// SimpleSupportedVSCurrencies /simple/supported_vs_currencies
func (c *Client) SimpleSupportedVSCurrencies() (*types.SimpleSupportedVSCurrencies, error) {
	url := fmt.Sprintf("%s/simple/supported_vs_currencies", baseURL)
	resp, err := c.MakeReq(url)
	if err != nil {
		return nil, err
	}
	var data *types.SimpleSupportedVSCurrencies
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// CoinsList /coins/list
func (c *Client) CoinsList() (*types.CoinList, error) {
	url := fmt.Sprintf("%s/coins/list", baseURL)
	resp, err := c.MakeReq(url)
	if err != nil {
		return nil, err
	}

	var data *types.CoinList
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// CoinsMarket /coins/market
func (c *Client) CoinsMarket(ctx context.Context, vsCurrency string, ids []string, order string, perPage int, page int, sparkline bool, priceChangePercentage []string) ([]*types.CoinsMarketItem, error) {
	if len(vsCurrency) == 0 {
		return nil, fmt.Errorf("vs_currency is required")
	}

	params := url.Values{}
	// vs_currency
	params.Add("vs_currency", vsCurrency)
	// order
	if len(order) == 0 {
		order = types.OrderTypeObject.MarketCapDesc
	}
	params.Add("order", order)
	// ids
	if len(ids) != 0 {
		idsParam := strings.Join(ids[:], ",")
		params.Add("ids", idsParam)
	}
	// per_page
	if perPage <= 0 || perPage > 250 {
		perPage = 100
	}

	params.Add("per_page", format.Int2String(perPage))
	params.Add("page", format.Int2String(page))
	// sparkline
	params.Add("sparkline", format.Bool2String(sparkline))
	// price_change_percentage
	if len(priceChangePercentage) != 0 {
		priceChangePercentageParam := strings.Join(priceChangePercentage[:], ",")
		params.Add("price_change_percentage", priceChangePercentageParam)
	}

	resp, err := c.MakeStreamingReq(ctx, buildUrlParams(baseURL, "/coins/markets?", params))
	if err != nil {
		return nil, errors.Wrap(err, "male streaming req")
	}
	defer resp.Close()

	res, err := unmarshalStreaming(resp, perPage)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshaling resp")
	}

	return res, nil
}

func unmarshalStreaming(reader io.ReadCloser, perPage int) ([]*types.CoinsMarketItem, error) {
	if perPage == 0 {
		return nil, nil
	}

	res := make([]*types.CoinsMarketItem, 0, perPage)

	decoder := json.NewDecoder(reader)
	_, err := decoder.Token()
	if err != nil {
		return nil, errors.Wrap(err, "invalid start token")
	}

	for decoder.More() {
		tempItem := &types.CoinsMarketItem{}
		err = decoder.Decode(tempItem)
		if err != nil {
			return nil, errors.Wrap(err, "decoding coins market item")
		}

		res = append(res, tempItem)
	}

	_, err = decoder.Token()
	if err != nil {
		return nil, errors.Wrap(err, "invalid end token")
	}

	return res, nil
}

func buildUrlParams(baseUrl, path string, params url.Values) string {
	sb := strings.Builder{}
	encodedParams := params.Encode()

	sb.Grow(len(baseUrl) + len(path) + len(encodedParams))

	sb.WriteString(baseUrl)
	sb.WriteString(path)
	sb.WriteString(encodedParams)

	return sb.String()
}

// CoinsID /coins/{id}
func (c *Client) CoinsID(id string, localization bool, tickers bool, marketData bool, communityData bool, developerData bool, sparkline bool) (*types.CoinsID, error) {

	if len(id) == 0 {
		return nil, fmt.Errorf("id is required")
	}
	params := url.Values{}
	params.Add("localization", format.Bool2String(sparkline))
	params.Add("tickers", format.Bool2String(tickers))
	params.Add("market_data", format.Bool2String(marketData))
	params.Add("community_data", format.Bool2String(communityData))
	params.Add("developer_data", format.Bool2String(developerData))
	params.Add("sparkline", format.Bool2String(sparkline))
	url := fmt.Sprintf("%s/coins/%s?%s", baseURL, id, params.Encode())
	resp, err := c.MakeReq(url)
	if err != nil {
		return nil, err
	}

	var data *types.CoinsID
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// CoinsIDTickers /coins/{id}/tickers
func (c *Client) CoinsIDTickers(id string, page int) (*types.CoinsIDTickers, error) {
	if len(id) == 0 {
		return nil, fmt.Errorf("id is required")
	}
	params := url.Values{}
	if page > 0 {
		params.Add("page", format.Int2String(page))
	}
	url := fmt.Sprintf("%s/coins/%s/tickers?%s", baseURL, id, params.Encode())
	resp, err := c.MakeReq(url)
	if err != nil {
		return nil, err
	}
	var data *types.CoinsIDTickers
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// CoinsIDHistory /coins/{id}/history?date={date}&localization=false
func (c *Client) CoinsIDHistory(id string, date string, localization bool) (*types.CoinsIDHistory, error) {
	if len(id) == 0 || len(date) == 0 {
		return nil, fmt.Errorf("id and date is required")
	}
	params := url.Values{}
	params.Add("date", date)
	params.Add("localization", format.Bool2String(localization))

	url := fmt.Sprintf("%s/coins/%s/history?%s", baseURL, id, params.Encode())
	resp, err := c.MakeReq(url)
	if err != nil {
		return nil, err
	}
	var data *types.CoinsIDHistory
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// CoinsIDMarketChart /coins/{id}/market_chart?vs_currency={usd, eur, jpy, etc.}&days={1,14,30,max}
func (c *Client) CoinsIDMarketChart(id string, vs_currency string, days string) (*types.CoinsIDMarketChart, error) {
	if len(id) == 0 || len(vs_currency) == 0 || len(days) == 0 {
		return nil, fmt.Errorf("id, vs_currency, and days is required")
	}

	params := url.Values{}
	params.Add("vs_currency", vs_currency)
	params.Add("days", days)

	url := fmt.Sprintf("%s/coins/%s/market_chart?%s", baseURL, id, params.Encode())
	resp, err := c.MakeReq(url)
	if err != nil {
		return nil, err
	}

	m := types.CoinsIDMarketChart{}
	err = json.Unmarshal(resp, &m)
	if err != nil {
		return &m, err
	}

	return &m, nil
}

// CoinsIDStatusUpdates

// CoinsIDContractAddress https://api.coingecko.com/api/v3/coins/{id}/contract/{contract_address}
// func CoinsIDContractAddress(id string, address string) (nil, error) {
// 	url := fmt.Sprintf("%s/coins/%s/contract/%s", baseURL, id, address)
// 	resp, err := request.MakeReq(url)
// 	if err != nil {
// 		return nil, err
// 	}
// }

// EventsCountries https://api.coingecko.com/api/v3/events/countries
func (c *Client) EventsCountries() ([]types.EventCountryItem, error) {
	url := fmt.Sprintf("%s/events/countries", baseURL)
	resp, err := c.MakeReq(url)
	if err != nil {
		return nil, err
	}
	var data *types.EventsCountries
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return data.Data, nil

}

// EventsTypes https://api.coingecko.com/api/v3/events/types
func (c *Client) EventsTypes() (*types.EventsTypes, error) {
	url := fmt.Sprintf("%s/events/types", baseURL)
	resp, err := c.MakeReq(url)
	if err != nil {
		return nil, err
	}
	var data *types.EventsTypes
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return data, nil

}

// ExchangeRates https://api.coingecko.com/api/v3/exchange_rates
func (c *Client) ExchangeRates() (*types.ExchangeRatesItem, error) {
	url := fmt.Sprintf("%s/exchange_rates", baseURL)
	resp, err := c.MakeReq(url)
	if err != nil {
		return nil, err
	}
	var data *types.ExchangeRatesResponse
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return &data.Rates, nil
}

// Global https://api.coingecko.com/api/v3/global
func (c *Client) Global() (*types.Global, error) {
	url := fmt.Sprintf("%s/global", baseURL)
	resp, err := c.MakeReq(url)
	if err != nil {
		return nil, err
	}
	var data *types.GlobalResponse
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return &data.Data, nil
}
