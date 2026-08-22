package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"goscrapy/internal/domtree"
	"goscrapy/internal/model"
	"goscrapy/internal/renderer"
)

func TestInferSelectorsKeepsFallbackForUniqueElement(t *testing.T) {
	tree, err := domtree.Parse([]byte(`<html><body><main><h1 class="headline">Release notes</h1><p>Details</p></main></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	var nodeID int
	for id, node := range tree.ByID {
		if node.Tag == "h1" {
			nodeID = id
			break
		}
	}
	if nodeID == 0 {
		t.Fatal("heading node not found")
	}

	snapshots := renderer.NewService("", nil)
	snapshots.Store().Put(&renderer.Record{
		ID:        "snap_unique",
		Tree:      tree,
		CreatedAt: time.Now(),
	})
	deps := &Deps{Snap: snapshots}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/snapshots/:id/selectors", deps.InferSelectors)
	body := strings.NewReader(`{"node_id":` + strconv.Itoa(nodeID) + `}`)
	req := httptest.NewRequest(http.MethodPost, "/snapshots/snap_unique/selectors", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data model.SelectorData `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.ListRule == nil {
		t.Fatalf("successful inference returned null list_rule: %s", rec.Body.String())
	}
	if envelope.Data.ListRule.FieldSelector != ".headline" || envelope.Data.ListRule.HitCount != 1 {
		t.Fatalf("fallback=%+v", envelope.Data.ListRule)
	}
}
