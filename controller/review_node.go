package controller

import (
	"strings"

	"just-vpn/model"
)

const reviewNodeName = "审核节点"

func configuredReviewNode(version string, code string) (model.Node, bool) {
	return reviewNodeFromConfig(
		version,
		code,
		model.ConfigValue(model.ConfigNodeReviewVersions, ""),
		model.ConfigValue(model.ConfigNodeReviewLink, ""),
	)
}

func reviewNodeFromConfig(version string, code string, versions string, link string) (model.Node, bool) {
	version = strings.TrimSpace(version)
	link = strings.TrimSpace(link)
	if version == "" || link == "" || !reviewVersionMatches(versions, version) {
		return model.Node{}, false
	}
	return model.Node{
		Name:     reviewNodeName,
		Code:     strings.ToUpper(strings.TrimSpace(code)),
		CodeName: reviewNodeName,
		LinkUrl:  link,
		Status:   model.NodeStatusEnabled,
	}, true
}

func reviewVersionMatches(versions string, version string) bool {
	version = strings.TrimSpace(version)
	if version == "" {
		return false
	}
	for _, configuredVersion := range strings.Split(versions, ",") {
		if strings.TrimSpace(configuredVersion) == version {
			return true
		}
	}
	return false
}
