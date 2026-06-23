package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"email-organizer/internal/models"
	"email-organizer/internal/store"
)

type RuleHandler struct {
	ruleRepo *store.RuleRepository
}

func NewRuleHandler(ruleRepo *store.RuleRepository) *RuleHandler {
	return &RuleHandler{ruleRepo: ruleRepo}
}

func (h *RuleHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.ruleRepo.List()
	if err != nil {
		Error(w, 500, err.Error())
		return
	}
	JSON(w, 200, rules)
}

func (h *RuleHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string              `json:"name"`
		Enabled  bool                `json:"enabled"`
		Trigger  string              `json:"trigger"`
		Cond     models.RuleCondition `json:"conditions"`
		Action   models.RuleAction   `json:"action"`
		Priority int                 `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, 400, "invalid json")
		return
	}
	condJSON, _ := json.Marshal(input.Cond)
	actionJSON, _ := json.Marshal(input.Action)

	rule := &models.Rule{
		Name:          input.Name,
		Enabled:       input.Enabled,
		Trigger:       input.Trigger,
		ConditionJSON: string(condJSON),
		ActionJSON:    string(actionJSON),
		Priority:      input.Priority,
	}
	id, err := h.ruleRepo.Create(rule)
	if err != nil {
		Error(w, 500, err.Error())
		return
	}
	rule.ID = id
	JSON(w, 201, rule)
}

func (h *RuleHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		Error(w, 400, "invalid rule id")
		return
	}
	rule, err := h.ruleRepo.GetByID(id)
	if err != nil {
		Error(w, 404, "rule not found")
		return
	}
	JSON(w, 200, rule)
}

func (h *RuleHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		Error(w, 400, "invalid rule id")
		return
	}
	var input struct {
		Name     string              `json:"name"`
		Enabled  bool                `json:"enabled"`
		Trigger  string              `json:"trigger"`
		Cond     models.RuleCondition `json:"conditions"`
		Action   models.RuleAction   `json:"action"`
		Priority int                 `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, 400, "invalid json")
		return
	}
	condJSON, _ := json.Marshal(input.Cond)
	actionJSON, _ := json.Marshal(input.Action)

	rule := &models.Rule{
		ID:            id,
		Name:          input.Name,
		Enabled:       input.Enabled,
		Trigger:       input.Trigger,
		ConditionJSON: string(condJSON),
		ActionJSON:    string(actionJSON),
		Priority:      input.Priority,
	}
	if err := h.ruleRepo.Update(rule); err != nil {
		Error(w, 500, err.Error())
		return
	}
	JSON(w, 200, rule)
}

func (h *RuleHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		Error(w, 400, "invalid rule id")
		return
	}
	if err := h.ruleRepo.Delete(id); err != nil {
		Error(w, 500, err.Error())
		return
	}
	JSON(w, 200, map[string]string{"status": "deleted"})
}

func (h *RuleHandler) ToggleRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		Error(w, 400, "invalid rule id")
		return
	}
	rule, err := h.ruleRepo.GetByID(id)
	if err != nil {
		Error(w, 404, "rule not found")
		return
	}
	if err := h.ruleRepo.SetEnabled(id, !rule.Enabled); err != nil {
		Error(w, 500, err.Error())
		return
	}
	JSON(w, 200, map[string]interface{}{"id": id, "enabled": !rule.Enabled})
}