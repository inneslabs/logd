package wp

import "testing"

func TestDecodeRequest(t *testing.T) {
	rawInput := `{
    "private_file": "",
    "client_site_id": "6062",
    "sent_vary": "Accept-Encoding",
    "timestamp": "23/Dec/2023:18:46:58 +0000",
    "body_bytes_sent": "81",
    "timestamp_iso8601": "2023-12-23T18:46:58+00:00",
    "http_host": "swissinfo-ch-develop.go-vip.net",
    "http_x_forwarded_for": "",
    "http_referer": "https://swissinfo-ch-develop.go-vip.net/eng/wp-admin/post.php?post=386276&action=edit",
    "remote_addr": "90.241.96.213",
    "http_version": "HTTP/2.0",
    "remote_user": "",
    "request_type": "POST",
    "request_url": "/eng/wp-admin/admin-ajax.php",
    "request_time": "0.626",
    "sent_cache_control": "no-cache, must-revalidate, max-age=0, no-store, private",
    "content_type": "application/json; charset=UTF-8",
    "status": "200",
    "upstream_country_code": "GB",
    "scheme": "https",
    "tls_version": "TLSv1.3",
    "ssl_client_verify": "NONE",
    "sent_x_cache": "miss",
    "wplogin": "usr_4c08fa0549f749f4bb9e67aefa63791c",
    "true_client_ip": "",
    "asn": "5378",
    "http_accept_language": "en-GB,en;q=0.5",
    "http_user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "request_id": "16e692df3900c0e8dbedbd5008cbbc2c"
	}`
	r, err := DecodeRequest(rawInput, "test")
	if err != nil {
		t.FailNow()
	}
	if r.Method != "POST" {
		t.Fatalf("expected Method == POST, got %s", r.Method)
	}
	if r.ResponseStatus != "200" {
		t.Fatalf("expected Status == 200, got %s", r.ResponseStatus)
	}
	if r.Url != "/eng/wp-admin/admin-ajax.php" {
		t.Fatalf("expected Status == /eng/wp-admin/admin-ajax.php, got %s", r.Url)
	}
}
