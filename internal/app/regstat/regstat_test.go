package regstat

import (
	"strings"
	"testing"
)

func TestProcessRegistryRequest(t *testing.T) {
	t.Run("empty body", func(t *testing.T) {
		wf := createMockWorkflow()
		s := server{workflow: wf}
		err := s.processRegistryRequest([]byte{})
		if err != nil {
			t.Errorf("expected nil err; got %s", err)
		}
		if len(*wf.receivedEvents) > 0 {
			t.Error("expected no events", wf.receivedEvents)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		wf := createMockWorkflow()
		s := server{workflow: wf}
		err := s.processRegistryRequest([]byte("abc"))
		if err == nil {
			t.Fatal("expected non nil err")
		}
		if !strings.Contains(err.Error(), "invalid character") {
			t.Errorf("expected invalid character; got %s", err)
		}
		if len(*wf.receivedEvents) > 0 {
			t.Error("expected no events", wf.receivedEvents)
		}
	})

	t.Run("unknown event", func(t *testing.T) {
		wf := createMockWorkflow()
		s := server{workflow: wf}
		err := s.processRegistryRequest([]byte("{\"events\":[{\"action\":\"boo\"}]}"))
		if err != nil {
			t.Errorf("expected nil err; got %s", err)
		}
		if len(*wf.receivedEvents) > 0 {
			t.Error("expected no events", wf.receivedEvents)
		}
	})

	t.Run("delete event", func(t *testing.T) {
		wf := createMockWorkflow()
		s := server{workflow: wf}
		err := s.processRegistryRequest([]byte("{\"events\":[{\"action\":\"delete\"}]}"))
		if err != nil {
			t.Errorf("expected nil err; got %s", err)
		}
		if len(*wf.receivedEvents) != 1 {
			t.Error("expected one event", wf.receivedEvents)
		}
	})

	t.Run("push event", func(t *testing.T) {
		wf := createMockWorkflow()
		s := server{workflow: wf}
		err := s.processRegistryRequest([]byte("{\"events\":[{\"action\":\"push\"}]}"))
		if err != nil {
			t.Errorf("expected nil err; got %s", err)
		}
		if len(*wf.receivedEvents) != 1 {
			t.Error("expected one event", wf.receivedEvents)
		}
	})

	t.Run("pull event", func(t *testing.T) {
		wf := createMockWorkflow()
		s := server{workflow: wf}
		err := s.processRegistryRequest([]byte("{\"events\":[{\"action\":\"pull\"}]}"))
		if err != nil {
			t.Errorf("expected nil err; got %s", err)
		}
		if len(*wf.receivedEvents) != 1 {
			t.Error("expected one event", wf.receivedEvents)
		}
	})
}
