package openapi2kong

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_pathKeySlug(t *testing.T) {
	tests := []struct {
		pathKey      string
		expected     string
		expectedInso string
	}{
		{"/", "~", ""}, // existing behaviour: the root path ends with a "/"
		{"/orders", "orders", "orders"},
		{"/orders/", "orders~", "orders"},
		{"/orders/{orderId}", "orders-orderid", "orders_orderId"},
		{"/orders/{orderId}/items/", "orders-orderid-items~", "orders_orderId_items"},
		{"/v1.0/Orders", "v1-0-orders", "v1_0_Orders"},
	}
	for _, tc := range tests {
		t.Run(tc.pathKey, func(t *testing.T) {
			assert.Equal(t, tc.expected, pathKeySlug(tc.pathKey, false))
			assert.Equal(t, tc.expectedInso, pathKeySlug(tc.pathKey, true))
		})
	}
}

func Test_joinName(t *testing.T) {
	assert.Equal(t, "doc_orders", joinName("doc", "orders", "_"))
	assert.Equal(t, "doc-orders", joinName("doc", "orders", "-"))
	assert.Equal(t, "doc", joinName("doc", "", "_"))
}

func Test_buildOperationBaseName(t *testing.T) {
	const (
		docName  = "doc"
		kongName = "My Name"
		opID     = "getOrder"
	)
	// the path base name, based on a path level x-kong-name "Single Order"
	pathBaseName := map[bool]string{false: "doc_single-order", true: "doc-Single_Order"}
	concat := map[bool]string{false: "_", true: "-"}

	tests := []struct {
		kongName       string
		operationID    string
		fromPathMethod bool
		expected       string
		expectedInso   string
	}{
		// existing naming: x-kong-name -> operationId -> path+method
		{kongName, opID, false, "doc_single-order_my-name", "doc-My_Name"},
		{kongName, "", false, "doc_single-order_my-name", "doc-My_Name"},
		{"", opID, false, "doc_getorder", "doc-getOrder"},
		{"", "", false, "doc_single-order_get", "doc-Single_Order-get"},
		// from path+method: always the actual path, and the method
		{kongName, opID, true, "doc_orders-id_get", "doc-orders_id-get"},
		{kongName, "", true, "doc_orders-id_get", "doc-orders_id-get"},
		{"", opID, true, "doc_orders-id_get", "doc-orders_id-get"},
		{"", "", true, "doc_orders-id_get", "doc-orders_id-get"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%q/%q/%v", tc.kongName, tc.operationID, tc.fromPathMethod), func(t *testing.T) {
			for _, inso := range []bool{false, true} {
				name := buildOperationBaseName(tc.kongName, tc.operationID, "GET", "/orders/{id}",
					docName, pathBaseName[inso], concat[inso], inso, tc.fromPathMethod)
				expected := tc.expected
				if inso {
					expected = tc.expectedInso
				}
				assert.Equal(t, expected, name, "inso=%v", inso)
			}
		})
	}

	t.Run("from path+method, root path", func(t *testing.T) {
		// same as existing naming for the root path without operationId
		assert.Equal(t, "doc_~_get", buildOperationBaseName("", opID, "GET", "/", docName, "doc", "_", false, true))
		assert.Equal(t, "doc_~_get", buildOperationBaseName("", "", "GET", "/", docName, "doc_~", "_", false, false))
		assert.Equal(t, "doc-get", buildOperationBaseName("", opID, "GET", "/", docName, "doc", "-", true, true))
	})
}

func Test_routeNameRegistry(t *testing.T) {
	r := make(routeNameRegistry)
	assert.NoError(t, r.register("a", "GET /a"))
	assert.NoError(t, r.register("b", "GET /b"))
	assert.EqualError(t, r.register("a", "GET /c"),
		"duplicate route name 'a' generated from path and method for operations 'GET /a' and 'GET /c'; "+
			"paths must remain unique after conversion to names")
}

// Test_Openapi2kong_RouteNameFromPathMethod tests the RouteNameFromPathMethod option end-to-end.
func Test_Openapi2kong_RouteNameFromPathMethod(t *testing.T) {
	spec := `
openapi: 3.0.3
info:
  title: orders
  version: v1
servers:
  - url: https://example.com
paths:
  /orders:
    get:
      operationId: listOrders
      responses:
        "200":
          description: OK
    post:
      x-kong-name: create
      operationId: createOrder
      responses:
        "200":
          description: OK
    put:
      x-kong-name: replace
      responses:
        "200":
          description: OK
    delete:
      responses:
        "200":
          description: OK
`
	routeNames := func(t *testing.T, opts O2kOptions) map[string]string {
		t.Helper()
		result, err := Convert([]byte(spec), opts)
		if !assert.NoError(t, err) {
			return nil
		}
		names := map[string]string{}
		for _, svc := range result["services"].([]interface{}) {
			for _, r := range svc.(map[string]interface{})["routes"].([]interface{}) {
				route := r.(map[string]interface{})
				names[route["methods"].([]string)[0]] = route["name"].(string)
			}
		}
		return names
	}

	t.Run("not set, existing naming", func(t *testing.T) {
		assert.Equal(t, map[string]string{
			"GET":    "orders_listorders",     // operationId
			"POST":   "orders_orders_create",  // x-kong-name wins over operationId
			"PUT":    "orders_orders_replace", // x-kong-name
			"DELETE": "orders_orders_delete",  // path+method
		}, routeNames(t, O2kOptions{}))
	})

	t.Run("not set, existing naming, inso compatible", func(t *testing.T) {
		assert.Equal(t, map[string]string{
			"GET":    "orders-listOrders",
			"POST":   "orders-create",
			"PUT":    "orders-replace",
			"DELETE": "orders-orders-delete",
		}, routeNames(t, O2kOptions{InsoCompat: true, SkipID: true}))
	})

	t.Run("set", func(t *testing.T) {
		assert.Equal(t, map[string]string{
			"GET":    "orders_orders_get",
			"POST":   "orders_orders_post",
			"PUT":    "orders_orders_put",
			"DELETE": "orders_orders_delete",
		}, routeNames(t, O2kOptions{RouteNameFromPathMethod: true}))
	})

	t.Run("set, inso compatible", func(t *testing.T) {
		assert.Equal(t, map[string]string{
			"GET":    "orders-orders-get",
			"POST":   "orders-orders-post",
			"PUT":    "orders-orders-put",
			"DELETE": "orders-orders-delete",
		}, routeNames(t, O2kOptions{RouteNameFromPathMethod: true, InsoCompat: true, SkipID: true}))
	})

	t.Run("route IDs follow the name", func(t *testing.T) {
		resultDefault, _ := Convert([]byte(spec), O2kOptions{})
		resultSet, _ := Convert([]byte(spec), O2kOptions{RouteNameFromPathMethod: true})
		routeIDs := func(result map[string]interface{}) map[string]string {
			ids := map[string]string{}
			svc := result["services"].([]interface{})[0].(map[string]interface{})
			for _, r := range svc["routes"].([]interface{}) {
				route := r.(map[string]interface{})
				ids[route["methods"].([]string)[0]] = route["id"].(string)
			}
			return ids
		}
		idsDefault, idsSet := routeIDs(resultDefault), routeIDs(resultSet)
		assert.NotEqual(t, idsDefault["GET"], idsSet["GET"], "name changed, so the ID changes")
		assert.Equal(t, idsDefault["DELETE"], idsSet["DELETE"], "name unchanged, so the ID is stable")
	})
}

func Test_Openapi2kong_RouteNameFromPathMethod_Duplicates(t *testing.T) {
	// "/orders/{order_id}" and "/orders/{order-id}" slugify to the same path name
	spec := `
openapi: 3.0.3
info:
  title: orders
  version: v1
servers:
  - url: https://example.com
paths:
  /orders/{order_id}:
    get:
      operationId: getOrderUnderscore
      responses:
        "200":
          description: OK
  /orders/{order-id}:
    get:
      operationId: getOrderDash
      responses:
        "200":
          description: OK
`
	t.Run("not set, unique by operationId", func(t *testing.T) {
		_, err := Convert([]byte(spec), O2kOptions{})
		assert.NoError(t, err)
	})

	t.Run("set, fails on the duplicate", func(t *testing.T) {
		_, err := Convert([]byte(spec), O2kOptions{RouteNameFromPathMethod: true})
		assert.EqualError(t, err, "duplicate route name 'orders_orders-order-id_get' generated from path and "+
			"method for operations 'GET /orders/{order-id}' and 'GET /orders/{order_id}'; "+
			"paths must remain unique after conversion to names")
	})

	t.Run("set, same path with and without trailing slash is unique", func(t *testing.T) {
		specSlash := `
openapi: 3.0.3
info:
  title: orders
  version: v1
paths:
  /orders:
    get:
      responses:
        "200":
          description: OK
  /orders/:
    get:
      responses:
        "200":
          description: OK
`
		_, err := Convert([]byte(specSlash), O2kOptions{RouteNameFromPathMethod: true})
		assert.NoError(t, err)
	})
}
