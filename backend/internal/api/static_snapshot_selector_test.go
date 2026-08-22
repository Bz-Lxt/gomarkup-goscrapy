package api_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"goscrapy/internal/api"
	"goscrapy/internal/renderer"
)

func TestStaticSnapshotReturnedNodeSupportsSelectors(t *testing.T) {
	capture, err := renderer.CaptureFromHTML(
		"https://shop.example/products/42",
		[]byte(`<!doctype html><html><body><main class="product-detail"><h1 data-product-name="42">Mechanical Keyboard</h1><p class="price">$129.00</p></main></body></html>`),
	)
	if err != nil {
		t.Fatalf("capture static snapshot: %v", err)
	}
	snapshots := renderer.NewService("", nil)
	snapshots.Store().Put(&renderer.Record{
		ID:        "snap_static_product",
		URL:       capture.URL,
		PNG:       capture.PNG,
		Tree:      capture.Tree,
		Nodes:     capture.Nodes,
		Source:    capture.Source,
		CreatedAt: time.Now(),
	})

	deps := &api.Deps{Snap: snapshots}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/snapshots/:id/image", deps.GetSnapshotImage)
	router.POST("/snapshots/:id/selectors", deps.InferSelectors)

	var headingID int
	for _, node := range capture.Nodes {
		if node.Tag == "h1" {
			headingID = node.NodeID
			break
		}
	}
	if headingID == 0 {
		t.Fatalf("snapshot response missing selectable heading: %+v", capture.Nodes)
	}

	image := httptest.NewRecorder()
	router.ServeHTTP(image, httptest.NewRequest(
		http.MethodGet,
		"/snapshots/snap_static_product/image",
		nil,
	))
	if image.Code != http.StatusOK || image.Body.Len() == 0 {
		t.Fatalf("read snapshot image: status=%d bytes=%d", image.Code, image.Body.Len())
	}

	infer := httptest.NewRecorder()
	inferRequest := httptest.NewRequest(
		http.MethodPost,
		"/snapshots/snap_static_product/selectors",
		bytes.NewBufferString(fmt.Sprintf(`{"node_id":%d}`, headingID)),
	)
	inferRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(infer, inferRequest)
	if infer.Code != http.StatusOK {
		t.Fatalf("infer selectors for returned node: status=%d body=%s", infer.Code, infer.Body.String())
	}
}
