package helper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// https://docs.claude.com/en/docs/build-with-claude/prompt-caching#1-hour-cache-duration
const claudeCacheCreation1hMultiplier = 6 / 3.75

func normalizeResolutionPriceKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func extractResolutionKeyFromTaskRequest(req relaycommon.TaskSubmitReq) string {
	candidates := []string{
		req.Resolution,
		req.ResolutionName,
		common.Interface2String(req.Metadata["output_resolution"]),
		common.Interface2String(req.Metadata["resolution"]),
		common.Interface2String(req.Metadata["resolution_name"]),
		common.Interface2String(req.Metadata["quality"]),
		req.Quality,
		req.Size,
	}
	for _, candidate := range candidates {
		if key := normalizeResolutionPriceKey(candidate); key != "" {
			return key
		}
	}
	return ""
}

func extractResolutionKeyFromRequest(request dto.Request) string {
	switch req := request.(type) {
	case *dto.GeneralOpenAIRequest:
		candidates := []string{
			req.OutputResolution,
			req.Resolution,
			req.Size,
		}
		if req.Quality != nil {
			candidates = append(candidates, *req.Quality)
		}
		for _, candidate := range candidates {
			if key := normalizeResolutionPriceKey(candidate); key != "" {
				return key
			}
		}
	case *dto.ImageRequest:
		for _, candidate := range []string{req.OutputResolution, req.Quality, req.Size} {
			if key := normalizeResolutionPriceKey(candidate); key != "" {
				return key
			}
		}
	}
	return ""
}

func extractPositiveIntValue(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		if v > 0 {
			return v, true
		}
	case int64:
		if v > 0 {
			return int(v), true
		}
	case float64:
		if v > 0 {
			return int(v), true
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && parsed > 0 {
			return parsed, true
		}
	}
	return 0, false
}

func extractSecondsFromRequestMetadata(metadata []byte) (int, bool) {
	if len(metadata) == 0 {
		return 0, false
	}
	var metadataMap map[string]any
	if err := common.Unmarshal(metadata, &metadataMap); err != nil {
		return 0, false
	}
	for _, key := range []string{"durationSeconds", "duration_seconds", "duration", "seconds"} {
		if seconds, ok := extractPositiveIntValue(metadataMap[key]); ok {
			return seconds, true
		}
	}
	return 0, false
}

func extractSecondsFromRequest(request dto.Request) (int, bool) {
	switch req := request.(type) {
	case *dto.GeneralOpenAIRequest:
		if seconds, ok := extractSecondsFromRequestMetadata(req.Metadata); ok {
			return seconds, true
		}
		if req.Duration != nil {
			if seconds, ok := extractPositiveIntValue(*req.Duration); ok {
				return seconds, true
			}
		}
		if req.Seconds != nil {
			if seconds, ok := extractPositiveIntValue(*req.Seconds); ok {
				return seconds, true
			}
		}
	}
	return 0, false
}

func extractSecondsFromTaskRequest(req relaycommon.TaskSubmitReq) (int, bool) {
	if req.Duration > 0 {
		return req.Duration, true
	}
	if seconds, ok := extractPositiveIntValue(req.Seconds); ok {
		return seconds, true
	}
	for _, key := range []string{"durationSeconds", "duration_seconds", "duration", "seconds"} {
		if seconds, ok := extractPositiveIntValue(req.Metadata[key]); ok {
			return seconds, true
		}
	}
	return 0, false
}

func GroupPriceCandidateGroups(info *relaycommon.RelayInfo) []string {
	if info == nil {
		return nil
	}
	seen := make(map[string]bool)
	groups := make([]string, 0, 2)
	for _, group := range []string{info.UserGroup, info.UsingGroup} {
		group = strings.TrimSpace(group)
		if group == "" || seen[group] {
			continue
		}
		seen[group] = true
		groups = append(groups, group)
	}
	return groups
}

func ResolveGroupModelPrice(info *relaycommon.RelayInfo) (float64, string, bool) {
	for _, group := range GroupPriceCandidateGroups(info) {
		if price, ok := ratio_setting.GetGroupModelPrice(group, info.OriginModelName); ok {
			return price, group, true
		}
	}
	return 0, "", false
}

func ResolveGroupModelPriceBySeconds(info *relaycommon.RelayInfo, seconds int) (float64, string, bool) {
	for _, group := range GroupPriceCandidateGroups(info) {
		if price, ok := ratio_setting.GetGroupModelPriceBySeconds(group, info.OriginModelName, seconds); ok {
			return price, group, true
		}
	}
	return 0, "", false
}

func ResolveGroupModelPriceBySecondsMin(info *relaycommon.RelayInfo) (float64, string, bool) {
	for _, group := range GroupPriceCandidateGroups(info) {
		if price, ok := ratio_setting.GetGroupModelPriceBySecondsMin(group, info.OriginModelName); ok {
			return price, group, true
		}
	}
	return 0, "", false
}

func ResolveGroupModelPriceByResolution(info *relaycommon.RelayInfo, resolution string) (float64, string, bool) {
	for _, group := range GroupPriceCandidateGroups(info) {
		if price, ok := ratio_setting.GetGroupModelPriceByResolution(group, info.OriginModelName, resolution); ok {
			return price, group, true
		}
	}
	return 0, "", false
}

func ResolveGroupModelPriceByResolutionMin(info *relaycommon.RelayInfo) (float64, string, bool) {
	for _, group := range GroupPriceCandidateGroups(info) {
		if price, ok := ratio_setting.GetGroupModelPriceByResolutionMin(group, info.OriginModelName); ok {
			return price, group, true
		}
	}
	return 0, "", false
}

func fixedPriceQuota(modelPrice float64, groupRatio float64, groupPriceOverride bool) int {
	if groupPriceOverride {
		return int(modelPrice * common.QuotaPerUnit)
	}
	return int(modelPrice * common.QuotaPerUnit * groupRatio)
}

// HandleGroupRatio checks for "auto_group" in the context and updates the group ratio and relayInfo.UsingGroup if present
func HandleGroupRatio(ctx *gin.Context, relayInfo *relaycommon.RelayInfo) types.GroupRatioInfo {
	groupRatioInfo := types.GroupRatioInfo{
		GroupRatio:        1.0, // default ratio
		GroupSpecialRatio: -1,
	}

	// check auto group
	if ctx != nil {
		if autoGroup, exists := ctx.Get("auto_group"); exists {
			logger.LogDebug(ctx, fmt.Sprintf("final group: %s", autoGroup))
			relayInfo.UsingGroup = autoGroup.(string)
		}
	}

	// check user group special ratio
	userGroupRatio, ok := ratio_setting.GetGroupGroupRatio(relayInfo.UserGroup, relayInfo.UsingGroup)
	if ok {
		// user group special ratio
		groupRatioInfo.GroupSpecialRatio = userGroupRatio
		groupRatioInfo.GroupRatio = userGroupRatio
		groupRatioInfo.HasSpecialRatio = true
	} else {
		// normal group ratio
		groupRatioInfo.GroupRatio = ratio_setting.GetGroupRatio(relayInfo.UsingGroup)
	}

	return groupRatioInfo
}

func ModelPriceHelper(c *gin.Context, info *relaycommon.RelayInfo, promptTokens int, meta *types.TokenCountMeta) (types.PriceData, error) {
	groupRatioInfo := HandleGroupRatio(c, info)
	seconds, _ := extractSecondsFromRequest(info.Request)
	resolution := extractResolutionKeyFromRequest(info.Request)
	price, usePrice, err := resolveConfiguredModelPrice(info, groupRatioInfo, seconds, resolution)
	if err != nil {
		return types.PriceData{}, err
	}
	if usePrice {
		info.PriceData = price
		return price, nil
	}

	preConsumedTokens := common.Max(promptTokens, common.PreConsumedQuota)
	if meta != nil && meta.MaxTokens != 0 {
		preConsumedTokens += meta.MaxTokens
	}
	modelRatio, configured, matchName := ratio_setting.GetModelRatio(info.OriginModelName)
	if !configured && !info.UserSetting.AcceptUnsetRatioModel {
		return types.PriceData{}, missingModelPriceError(matchName)
	}
	preConsumedQuota := int(float64(preConsumedTokens) * modelRatio * groupRatioInfo.GroupRatio)
	cacheRatio, _ := ratio_setting.GetCacheRatio(info.OriginModelName)
	cacheCreationRatio, _ := ratio_setting.GetCreateCacheRatio(info.OriginModelName)
	imageRatio, _ := ratio_setting.GetImageRatio(info.OriginModelName)
	freeModel := !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume &&
		(groupRatioInfo.GroupRatio == 0 || modelRatio == 0)
	priceData := types.PriceData{
		FreeModel: freeModel, ModelRatio: modelRatio, BillingType: "per-token",
		GroupRatioInfo: groupRatioInfo, QuotaToPreConsume: preConsumedQuota,
		CompletionRatio: ratio_setting.GetCompletionRatio(info.OriginModelName),
		CacheRatio:      cacheRatio, ImageRatio: imageRatio,
		AudioRatio:           ratio_setting.GetAudioRatio(info.OriginModelName),
		AudioCompletionRatio: ratio_setting.GetAudioCompletionRatio(info.OriginModelName),
		CacheCreationRatio:   cacheCreationRatio, CacheCreation5mRatio: cacheCreationRatio,
		CacheCreation1hRatio: cacheCreationRatio * claudeCacheCreation1hMultiplier,
	}

	if common.DebugEnabled {
		println(fmt.Sprintf("model_price_helper result: %s", priceData.ToSetting()))
	}
	info.PriceData = priceData
	return priceData, nil
}

func missingModelPriceError(name string) error {
	return fmt.Errorf("模型 %s 倍率或价格未配置，请联系管理员设置或开始自用模式；Model %s ratio or price not set, please set or start self-use mode", name, name)
}

// resolveConfiguredModelPrice selects a tariff, not a model capability. Model
// metadata may supply request defaults but never selects the billing mode.
func resolveConfiguredModelPrice(info *relaycommon.RelayInfo, groupRatioInfo types.GroupRatioInfo, seconds int, resolution string) (types.PriceData, bool, error) {
	name := info.OriginModelName
	if spec, ok := common.GetFa2ImageModelSpec(name); ok && (resolution == "" || resolution == "standard" || resolution == "auto") {
		resolution = spec.DefaultResolution
	}
	if seconds <= 0 && common.Is933VideoModel(name) {
		seconds = common.Video933DefaultDuration
	}
	if resolution == "" {
		switch {
		case common.Is933VideoModel(name):
			resolution = common.Video933Resolution(name)
		case common.IsVideo25Model(name):
			resolution = common.Video25Resolution(name)
		}
	}

	// A group tariff takes priority. Within each scope, use the same order as
	// the pricing editor: duration, resolution, then a fixed per-request price.
	for _, group := range append(GroupPriceCandidateGroups(info), "") {
		var mode string
		var price float64
		var ok bool
		if group == "" {
			if _, configured := ratio_setting.GetModelPriceBySecondsMap(name); configured {
				mode = "duration"
				price, ok = ratio_setting.GetModelPriceBySeconds(name, seconds)
			} else if _, configured := ratio_setting.GetModelPriceByResolutionMap(name); configured {
				mode = "resolution"
				price, ok = ratio_setting.GetModelPriceByResolution(name, resolution)
			} else if price, ok = ratio_setting.GetModelPrice(name, false); ok {
				mode = "per-request"
			}
		} else {
			if _, configured := ratio_setting.GetGroupModelPriceBySecondsMap(group, name); configured {
				mode = "duration"
				price, ok = ratio_setting.GetGroupModelPriceBySeconds(group, name, seconds)
			} else if _, configured := ratio_setting.GetGroupModelPriceByResolutionMap(group, name); configured {
				mode = "resolution"
				price, ok = ratio_setting.GetGroupModelPriceByResolution(group, name, resolution)
			} else if price, ok = ratio_setting.GetGroupModelPrice(group, name); ok {
				mode = "per-request"
			}
		}
		if mode == "" {
			continue
		}
		if !ok || price < 0 {
			return types.PriceData{}, true, fmt.Errorf("%s has no applicable %s price for %d seconds at %s", name, mode, seconds, resolution)
		}
		quota := fixedPriceQuota(price, groupRatioInfo.GroupRatio, group != "")
		result := types.PriceData{
			ModelPrice: price, Quota: quota, BaseQuota: quota, QuotaToPreConsume: quota, UsePrice: true,
			GroupRatioInfo: groupRatioInfo, GroupPriceOverride: group != "",
			GroupPriceOverrideGroup: group, BillingType: mode,
			FreeModel: !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume && (price == 0 || (group == "" && groupRatioInfo.GroupRatio == 0)),
		}
		if mode == "duration" {
			result.BillingSeconds = seconds
			if common.QuotaPerUnit > 0 {
				result.BillingTotalPrice = float64(quota) / common.QuotaPerUnit
				result.BillingUnitPrice = result.BillingTotalPrice / float64(seconds)
			}
		}
		return result, true, nil
	}
	return types.PriceData{}, false, nil
}

// ModelPriceHelperPerCall resolves a task tariff. Adaptor-provided duration
// is a usage measurement, not a multiplier for a fixed or resolution price.
func ModelPriceHelperPerCall(c *gin.Context, info *relaycommon.RelayInfo, usageSeconds ...int) (types.PriceData, error) {
	groupRatioInfo := HandleGroupRatio(c, info)
	seconds, _ := extractSecondsFromRequest(info.Request)
	resolution := extractResolutionKeyFromRequest(info.Request)
	if c != nil {
		if req, err := relaycommon.GetTaskRequest(c); err == nil {
			if value, ok := extractSecondsFromTaskRequest(req); ok {
				seconds = value
			}
			if resolution == "" {
				resolution = extractResolutionKeyFromTaskRequest(req)
			}
		}
	}
	if len(usageSeconds) != 0 && usageSeconds[0] > 0 {
		seconds = usageSeconds[0]
	}
	if price, found, err := resolveConfiguredModelPrice(info, groupRatioInfo, seconds, resolution); found || err != nil {
		return price, err
	}

	modelRatio, configured, matchName := ratio_setting.GetModelRatio(info.OriginModelName)
	if !configured && !info.UserSetting.AcceptUnsetRatioModel {
		return types.PriceData{}, missingModelPriceError(matchName)
	}
	quota := int(float64(common.PreConsumedQuota) * modelRatio * groupRatioInfo.GroupRatio)
	return types.PriceData{
		ModelRatio: modelRatio, BillingType: "per-token", GroupRatioInfo: groupRatioInfo,
		Quota: quota, BaseQuota: quota,
		FreeModel: !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume &&
			(modelRatio == 0 || groupRatioInfo.GroupRatio == 0),
	}, nil
}

func ContainPriceOrRatio(modelName string) bool {
	_, _, ok := ratio_setting.GetModelRatioOrPrice(modelName)
	return ok
}
