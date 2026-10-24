package server_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestCreateRenameAndListPersons(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()

	child := c.mustStatus(http.StatusCreated, "POST", "/api/persons", map[string]any{"name": " Kim "})
	if child.Body["name"] != "Kim" || child.Body["isUser"] != false {
		t.Fatalf("created person: %s", child.Raw)
	}
	id := child.Body["id"]
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/persons/%v", id), map[string]any{"name": "Kimi"})

	list := c.mustStatus(http.StatusOK, "GET", "/api/persons", nil)
	persons := list.Body["persons"].([]any)
	names := map[string]bool{}
	for _, p := range persons {
		p := p.(map[string]any)
		names[fmt.Sprintf("%v:%v", p["name"], p["isUser"])] = true
	}
	if len(persons) != 2 || !names["Alex:true"] || !names["Kimi:false"] {
		t.Fatalf("persons: %s", list.Raw)
	}
}

func TestPersonWithoutNameIsRejected(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	res := c.mustStatus(http.StatusBadRequest, "POST", "/api/persons", map[string]any{"name": "  "})
	if res.errorCode() != "name_missing" {
		t.Fatalf("error code %q", res.errorCode())
	}
}

func TestPersonWithoutItemsCanBeDeleted(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	id := c.createPerson("Kim")

	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/persons/%d", id), nil)
	res := c.mustStatus(http.StatusNotFound, "PUT", fmt.Sprintf("/api/persons/%d", id), map[string]any{"name": "Kim"})
	if res.errorCode() != "not_found" {
		t.Fatalf("error code %q", res.errorCode())
	}
}

func TestPersonWithUserAccessCannotBeDeleted(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	res := c.mustStatus(http.StatusConflict, "DELETE", fmt.Sprintf("/api/persons/%d", c.me()), nil)
	if res.errorCode() != "has_user_access" {
		t.Fatalf("error code %q", res.errorCode())
	}
}

func TestLastOwnerOfAnItemCannotBeDeleted(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	kim := c.createPerson("Kim")
	item := c.createItem("Kuscheltier", []int64{kim}, testJPEG(t))

	res := c.mustStatus(http.StatusConflict, "DELETE", fmt.Sprintf("/api/persons/%d", kim), nil)
	if res.errorCode() != "last_owner" {
		t.Fatalf("error code %q", res.errorCode())
	}

	// With a second owner Kim is no longer the last owner; deleting only removes the ownership.
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/items/%v/owners/%d", item.Body["id"], c.me()), nil)
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/persons/%d", kim), nil)
	after := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/items/%v", item.Body["id"]), nil)
	if got := ownerNames(after); len(got) != 1 || got[0] != "Alex" {
		t.Fatalf("owners after deleting: %v", got)
	}
}
