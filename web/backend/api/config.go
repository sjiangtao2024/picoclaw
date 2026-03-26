package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// registerConfigRoutes binds configuration management endpoints to the ServeMux.
func (h *Handler) registerConfigRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/config", h.handleGetConfig)
	mux.HandleFunc("PUT /api/config", h.handleUpdateConfig)
	mux.HandleFunc("PATCH /api/config", h.handlePatchConfig)
	mux.HandleFunc("POST /api/config/test-command-patterns", h.handleTestCommandPatterns)
}

// handleGetConfig returns the complete system configuration.
//
//	GET /api/config
func (h *Handler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	payload, err := configPayloadWithSecurityHints(cfg)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleUpdateConfig updates the complete system configuration.
//
//	PUT /api/config
func (h *Handler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var cfg config.Config
	if err = json.Unmarshal(body, &cfg); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if execAllowRemoteOmitted(body) {
		cfg.Tools.Exec.AllowRemote = config.DefaultConfig().Tools.Exec.AllowRemote
	}

	// Load existing config and copy security credentials before validation,
	// so that security-managed fields (e.g. pico token) are available.
	oldCfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	cfg.SecurityCopyFrom(oldCfg)
	applyConfigSecurityPatch(&cfg, body)

	if errs := validateConfig(&cfg); len(errs) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "validation_error",
			"errors": errs,
		})
		return
	}

	logger.Infof("configuration updated successfully")

	if err := config.SaveConfig(h.configPath, &cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func execAllowRemoteOmitted(body []byte) bool {
	var raw struct {
		Tools *struct {
			Exec *struct {
				AllowRemote *bool `json:"allow_remote"`
			} `json:"exec"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return false
	}
	return raw.Tools == nil || raw.Tools.Exec == nil || raw.Tools.Exec.AllowRemote == nil
}

// handlePatchConfig partially updates the system configuration using JSON Merge Patch (RFC 7396).
// Only the fields present in the request body will be updated; all other fields remain unchanged.
//
//	PATCH /api/config
func (h *Handler) handlePatchConfig(w http.ResponseWriter, r *http.Request) {
	patchBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate the patch is valid JSON
	var patch map[string]any
	if err = json.Unmarshal(patchBody, &patch); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Load existing config and marshal to a map for merging
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	existing, err := json.Marshal(cfg)
	if err != nil {
		http.Error(w, "Failed to serialize current config", http.StatusInternalServerError)
		return
	}

	var base map[string]any
	if err = json.Unmarshal(existing, &base); err != nil {
		http.Error(w, "Failed to parse current config", http.StatusInternalServerError)
		return
	}

	// Recursively merge patch into base
	mergeMap(base, patch)

	// Convert merged map back to Config struct
	merged, err := json.Marshal(base)
	if err != nil {
		http.Error(w, "Failed to serialize merged config", http.StatusInternalServerError)
		return
	}

	var newCfg config.Config
	if err := json.Unmarshal(merged, &newCfg); err != nil {
		http.Error(w, fmt.Sprintf("Merged config is invalid: %v", err), http.StatusBadRequest)
		return
	}

	// Restore security fields (tokens/keys) from the loaded config before validation,
	// because private fields are lost during JSON round-trip.
	newCfg.SecurityCopyFrom(cfg)
	if err := newCfg.ApplySecurity(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply security config: %v", err), http.StatusInternalServerError)
		return
	}
	applyConfigSecurityPatch(&newCfg, patchBody)

	if errs := validateConfig(&newCfg); len(errs) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "validation_error",
			"errors": errs,
		})
		return
	}

	if err := config.SaveConfig(h.configPath, &newCfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func configPayloadWithSecurityHints(cfg *config.Config) (map[string]any, error) {
	body, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	channels := nestedMap(payload, "channels")
	if feishu := nestedMap(channels, "feishu"); feishu != nil {
		feishu["app_secret_set"] = cfg.Channels.Feishu.AppSecret() != ""
		feishu["encrypt_key_set"] = cfg.Channels.Feishu.EncryptKey() != ""
		feishu["verification_token_set"] = cfg.Channels.Feishu.VerificationToken() != ""
	}
	if qq := nestedMap(channels, "qq"); qq != nil {
		qq["app_secret_set"] = cfg.Channels.QQ.AppSecret() != ""
	}
	if dingtalk := nestedMap(channels, "dingtalk"); dingtalk != nil {
		dingtalk["client_secret_set"] = cfg.Channels.DingTalk.ClientSecret() != ""
	}
	if slack := nestedMap(channels, "slack"); slack != nil {
		slack["bot_token_set"] = cfg.Channels.Slack.BotToken() != ""
		slack["app_token_set"] = cfg.Channels.Slack.AppToken() != ""
	}
	if line := nestedMap(channels, "line"); line != nil {
		line["channel_secret_set"] = cfg.Channels.LINE.ChannelSecret() != ""
		line["channel_access_token_set"] = cfg.Channels.LINE.ChannelAccessToken() != ""
	}
	if onebot := nestedMap(channels, "onebot"); onebot != nil {
		onebot["access_token_set"] = cfg.Channels.OneBot.AccessToken() != ""
	}
	if wecom := nestedMap(channels, "wecom"); wecom != nil {
		wecom["secret_set"] = cfg.Channels.WeCom.Secret() != ""
	}
	if telegram := nestedMap(channels, "telegram"); telegram != nil {
		telegram["token_set"] = cfg.Channels.Telegram.Token() != ""
	}
	if discord := nestedMap(channels, "discord"); discord != nil {
		discord["token_set"] = cfg.Channels.Discord.Token() != ""
	}
	if matrix := nestedMap(channels, "matrix"); matrix != nil {
		matrix["access_token_set"] = cfg.Channels.Matrix.AccessToken() != ""
	}
	if pico := nestedMap(channels, "pico"); pico != nil {
		pico["token_set"] = cfg.Channels.Pico.Token() != ""
	}
	tools := nestedMap(payload, "tools")
	web := nestedMap(tools, "web")
	if baiduSearch := nestedMap(web, "baidu_search"); baiduSearch != nil {
		baiduSearch["api_key_set"] = cfg.Tools.Web.BaiduSearch.APIKey() != ""
	}

	return payload, nil
}

func nestedMap(parent map[string]any, key string) map[string]any {
	if parent == nil {
		return nil
	}
	child, ok := parent[key].(map[string]any)
	if !ok {
		return nil
	}
	return child
}

func applyConfigSecurityPatch(cfg *config.Config, rawJSON []byte) {
	var patch map[string]any
	if err := json.Unmarshal(rawJSON, &patch); err != nil {
		return
	}
	if channels, ok := patch["channels"].(map[string]any); ok {
		if feishu, ok := channels["feishu"].(map[string]any); ok {
			if v, ok := feishu["app_secret"].(string); ok {
				cfg.Channels.Feishu.SetAppSecret(v)
			}
			if v, ok := feishu["encrypt_key"].(string); ok {
				cfg.Channels.Feishu.SetEncryptKey(v)
			}
			if v, ok := feishu["verification_token"].(string); ok {
				cfg.Channels.Feishu.SetVerificationToken(v)
			}
		}
		if qq, ok := channels["qq"].(map[string]any); ok {
			if v, ok := qq["app_secret"].(string); ok {
				cfg.Channels.QQ.SetAppSecret(v)
			}
		}
		if dingtalk, ok := channels["dingtalk"].(map[string]any); ok {
			if v, ok := dingtalk["client_secret"].(string); ok {
				cfg.Channels.DingTalk.SetClientSecret(v)
			}
		}
		if slack, ok := channels["slack"].(map[string]any); ok {
			if v, ok := slack["bot_token"].(string); ok {
				cfg.Channels.Slack.SetBotToken(v)
			}
			if v, ok := slack["app_token"].(string); ok {
				cfg.Channels.Slack.SetAppToken(v)
			}
		}
		if line, ok := channels["line"].(map[string]any); ok {
			if v, ok := line["channel_secret"].(string); ok {
				cfg.Channels.LINE.SetChannelSecret(v)
			}
			if v, ok := line["channel_access_token"].(string); ok {
				cfg.Channels.LINE.SetChannelAccessToken(v)
			}
		}
		if onebot, ok := channels["onebot"].(map[string]any); ok {
			if v, ok := onebot["access_token"].(string); ok {
				cfg.Channels.OneBot.SetAccessToken(v)
			}
		}
		if wecom, ok := channels["wecom"].(map[string]any); ok {
			if v, ok := wecom["secret"].(string); ok {
				cfg.Channels.WeCom.SetSecret(v)
			}
		}
		if telegram, ok := channels["telegram"].(map[string]any); ok {
			if v, ok := telegram["token"].(string); ok {
				cfg.Channels.Telegram.SetToken(v)
			}
		}
		if discord, ok := channels["discord"].(map[string]any); ok {
			if v, ok := discord["token"].(string); ok {
				cfg.Channels.Discord.SetToken(v)
			}
		}
		if matrix, ok := channels["matrix"].(map[string]any); ok {
			if v, ok := matrix["access_token"].(string); ok {
				cfg.Channels.Matrix.SetAccessToken(v)
			}
		}
		if pico, ok := channels["pico"].(map[string]any); ok {
			if v, ok := pico["token"].(string); ok {
				cfg.Channels.Pico.SetToken(v)
			}
		}
	}
	if tools, ok := patch["tools"].(map[string]any); ok {
		if web, ok := tools["web"].(map[string]any); ok {
			if baidu, ok := web["baidu_search"].(map[string]any); ok {
				if v, ok := baidu["api_key"].(string); ok {
					cfg.Tools.Web.BaiduSearch.SetAPIKey(v)
				}
			}
		}
	}
}

// handleTestCommandPatterns tests a command against whitelist and blacklist patterns.
//
//	POST /api/config/test-command-patterns
func (h *Handler) handleTestCommandPatterns(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		AllowPatterns []string `json:"allow_patterns"`
		DenyPatterns  []string `json:"deny_patterns"`
		Command       string   `json:"command"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	lower := strings.ToLower(strings.TrimSpace(req.Command))

	type result struct {
		Allowed          bool    `json:"allowed"`
		Blocked          bool    `json:"blocked"`
		MatchedWhitelist *string `json:"matched_whitelist,omitempty"`
		MatchedBlacklist *string `json:"matched_blacklist,omitempty"`
	}

	resp := result{Allowed: false, Blocked: false}

	// Check whitelist first
	for _, pattern := range req.AllowPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue // skip invalid patterns
		}
		if re.MatchString(lower) {
			resp.Allowed = true
			resp.MatchedWhitelist = &pattern
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	}

	// Check blacklist
	for _, pattern := range req.DenyPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(lower) {
			resp.Blocked = true
			resp.MatchedBlacklist = &pattern
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// validateConfig checks the config for common errors before saving.
// Returns a list of human-readable error strings; empty means valid.
func validateConfig(cfg *config.Config) []string {
	var errs []string

	// Validate model_list entries
	if err := cfg.ValidateModelList(); err != nil {
		errs = append(errs, err.Error())
	}

	// Gateway port range
	if cfg.Gateway.Port != 0 && (cfg.Gateway.Port < 1 || cfg.Gateway.Port > 65535) {
		errs = append(errs, fmt.Sprintf("gateway.port %d is out of valid range (1-65535)", cfg.Gateway.Port))
	}

	// Pico channel: token required when enabled
	if cfg.Channels.Pico.Enabled && cfg.Channels.Pico.Token() == "" {
		errs = append(errs, "channels.pico.token is required when pico channel is enabled")
	}

	// Telegram: token required when enabled
	if cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token() == "" {
		errs = append(errs, "channels.telegram.token is required when telegram channel is enabled")
	}

	// Discord: token required when enabled
	if cfg.Channels.Discord.Enabled && cfg.Channels.Discord.Token() == "" {
		errs = append(errs, "channels.discord.token is required when discord channel is enabled")
	}

	if cfg.Channels.WeCom.Enabled {
		if cfg.Channels.WeCom.BotID == "" {
			errs = append(errs, "channels.wecom.bot_id is required when wecom channel is enabled")
		}
		if cfg.Channels.WeCom.Secret() == "" {
			errs = append(errs, "channels.wecom.secret is required when wecom channel is enabled")
		}
	}

	if cfg.Tools.Exec.Enabled {
		if cfg.Tools.Exec.EnableDenyPatterns {
			errs = append(
				errs,
				validateRegexPatterns("tools.exec.custom_deny_patterns", cfg.Tools.Exec.CustomDenyPatterns)...)
		}
		errs = append(
			errs,
			validateRegexPatterns("tools.exec.custom_allow_patterns", cfg.Tools.Exec.CustomAllowPatterns)...)
	}

	return errs
}

func validateRegexPatterns(field string, patterns []string) []string {
	var errs []string
	for index, pattern := range patterns {
		if _, err := regexp.Compile(pattern); err != nil {
			errs = append(errs, fmt.Sprintf("%s[%d] is not a valid regular expression: %v", field, index, err))
		}
	}
	return errs
}

// mergeMap recursively merges src into dst (JSON Merge Patch semantics).
// - If a key in src has a null value, it is deleted from dst.
// - If both dst and src have a nested object for the same key, merge recursively.
// - Otherwise the value from src overwrites dst.
func mergeMap(dst, src map[string]any) {
	for key, srcVal := range src {
		if srcVal == nil {
			delete(dst, key)
			continue
		}
		srcMap, srcIsMap := srcVal.(map[string]any)
		dstMap, dstIsMap := dst[key].(map[string]any)
		if srcIsMap && dstIsMap {
			mergeMap(dstMap, srcMap)
		} else {
			dst[key] = srcVal
		}
	}
}
