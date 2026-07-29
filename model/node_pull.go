package model

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	nodeSubscriptionTimeout  = 30 * time.Second
	nodeSubscriptionMaxBytes = 10 << 20
	nodeSubscriptionCode     = "AUTO"
	nodeSubscriptionName     = "智能线路"
	nodeSubscriptionMinNodes = 2
)

// NodeSubscriptionSyncResult 节点订阅同步结果
type NodeSubscriptionSyncResult struct {
	Skipped   bool
	Pulled    int
	Added     int
	Activated int
	Retained  bool
}

type nodeSubscriptionItem struct {
	IP       string `json:"ip"`
	Content  string `json:"content"`
	Code     string `json:"code"`
	CodeName string `json:"code_name"`
	NodeType string `json:"node_type"`
}

type nodeDispatchPullResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Nodes []nodeSubscriptionItem `json:"nodes"`
	} `json:"data"`
}

// SyncNodesFromConfiguredSource 拉取订阅节点；少于两个有效节点时保留已有启用节点
func SyncNodesFromConfiguredSource() (NodeSubscriptionSyncResult, error) {
	pullURL := strings.TrimSpace(ConfigValue(ConfigNodePullURL, ""))
	if pullURL == "" {
		return NodeSubscriptionSyncResult{Skipped: true}, nil
	}

	nodes, err := fetchNodeSubscription(pullURL)
	if err != nil {
		return NodeSubscriptionSyncResult{}, err
	}
	return syncPulledNodes(nodes)
}

func fetchNodeSubscription(subscriptionURL string) ([]nodeSubscriptionItem, error) {
	request, err := http.NewRequest(http.MethodGet, subscriptionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create node subscription request: %w", err)
	}

	response, err := (&http.Client{Timeout: nodeSubscriptionTimeout}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("request node subscription: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("node subscription http status %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, nodeSubscriptionMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read node subscription: %w", err)
	}
	if len(body) > nodeSubscriptionMaxBytes {
		return nil, fmt.Errorf("node subscription response exceeds %d bytes", nodeSubscriptionMaxBytes)
	}

	nodes, err := parseNodeSubscriptionResponse(body)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("node subscription has no valid nodes")
	}
	return nodes, nil
}

func parseNodeSubscriptionResponse(body []byte) ([]nodeSubscriptionItem, error) {
	content := strings.TrimSpace(string(body))
	if strings.HasPrefix(content, "{") {
		var payload nodeDispatchPullResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("decode node dispatch response: %w", err)
		}
		if payload.Code != http.StatusOK {
			return nil, fmt.Errorf("node dispatch response failed: %s", payload.Message)
		}
		return normalizeNodeSubscriptionItems(payload.Data.Nodes), nil
	}

	links := parseNodeSubscription(content)
	nodes := make([]nodeSubscriptionItem, 0, len(links))
	for _, link := range links {
		nodeType, address, _ := nodeSubscriptionMetadata(link)
		if nodeType == "" || address == "" || nodeType == "vmess" {
			continue
		}
		nodes = append(nodes, nodeSubscriptionItem{
			IP:       address,
			Content:  link,
			Code:     nodeSubscriptionCode,
			CodeName: nodeSubscriptionName,
			NodeType: nodeType,
		})
	}
	return normalizeNodeSubscriptionItems(nodes), nil
}

func normalizeNodeSubscriptionItems(items []nodeSubscriptionItem) []nodeSubscriptionItem {
	nodes := make([]nodeSubscriptionItem, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item.Content = strings.TrimSpace(item.Content)
		item.IP = strings.TrimSpace(item.IP)
		if item.Content == "" || !strings.Contains(item.Content, "://") || item.IP == "" {
			continue
		}
		item.Code = strings.ToUpper(strings.TrimSpace(item.Code))
		if item.Code == "" {
			item.Code = nodeSubscriptionCode
		}
		item.CodeName = strings.TrimSpace(item.CodeName)
		if item.CodeName == "" {
			item.CodeName = nodeSubscriptionName
		}
		item.NodeType = strings.TrimSpace(item.NodeType)
		if item.NodeType == "" && strings.Contains(item.Content, "://") {
			item.NodeType, _, _ = nodeSubscriptionMetadata(item.Content)
		}
		if item.NodeType == "" {
			continue
		}
		if _, exists := seen[item.Content]; exists {
			continue
		}
		seen[item.Content] = struct{}{}
		nodes = append(nodes, item)
	}
	return nodes
}

func parseNodeSubscription(content string) []string {
	lines := strings.Split(content, "\n")
	links := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		link := strings.TrimSpace(line)
		if link == "" || strings.HasPrefix(link, "#") || !strings.Contains(link, "://") {
			continue
		}
		if _, exists := seen[link]; exists {
			continue
		}
		seen[link] = struct{}{}
		links = append(links, link)
	}
	return links
}

func syncPulledNodes(nodes []nodeSubscriptionItem) (NodeSubscriptionSyncResult, error) {
	result := NodeSubscriptionSyncResult{Pulled: len(nodes)}
	newNodes := make([]Node, 0)
	links := make([]string, 0, len(nodes))
	nodeByLink := make(map[string]nodeSubscriptionItem, len(nodes))
	for _, node := range nodes {
		links = append(links, node.Content)
		nodeByLink[node.Content] = node
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		var existing []Node
		if err := tx.Where("link_url IN ?", links).Find(&existing).Error; err != nil {
			return err
		}

		existingLinks := make(map[string]struct{}, len(existing))
		for _, node := range existing {
			existingLinks[node.LinkUrl] = struct{}{}
			item := nodeByLink[node.LinkUrl]
			name := nodeSubscriptionDisplayName(item)
			if node.Code == item.Code && node.CodeName == item.CodeName && node.NodeType == item.NodeType && node.Address == item.IP && node.Name == name {
				continue
			}
			if err := tx.Model(&Node{}).Where("id = ?", node.Id).Updates(map[string]interface{}{
				"code":      item.Code,
				"code_name": item.CodeName,
				"node_type": item.NodeType,
				"address":   item.IP,
				"name":      name,
			}).Error; err != nil {
				return err
			}
		}
		for _, item := range nodes {
			if _, exists := existingLinks[item.Content]; exists {
				continue
			}
			name := nodeSubscriptionDisplayName(item)
			newNodes = append(newNodes, Node{
				Code:     item.Code,
				CodeName: item.CodeName,
				Name:     name,
				NodeType: item.NodeType,
				LinkUrl:  item.Content,
				Address:  item.IP,
				Status:   NodeStatusEnabled,
			})
		}

		if shouldReplacePulledNodes(len(nodes)) {
			if err := tx.Model(&Node{}).Where("status <> ?", NodeStatusDisabled).Update("status", NodeStatusDisabled).Error; err != nil {
				return err
			}
		} else {
			result.Retained = true
		}
		if err := tx.Model(&Node{}).Where("link_url IN ?", links).Update("status", NodeStatusEnabled).Error; err != nil {
			return err
		}
		if len(newNodes) > 0 {
			if err := tx.CreateInBatches(newNodes, 200).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return NodeSubscriptionSyncResult{}, fmt.Errorf("sync node subscription: %w", err)
	}

	InvalidateAvailableNodeCache()
	result.Added = len(newNodes)
	result.Activated = len(links)
	return result, nil
}

func shouldReplacePulledNodes(nodeCount int) bool {
	return nodeCount >= nodeSubscriptionMinNodes
}

func nodeSubscriptionMetadata(link string) (nodeType, address, name string) {
	parsed, err := url.Parse(link)
	if err == nil {
		nodeType = strings.ToLower(parsed.Scheme)
		name = strings.TrimSpace(parsed.Fragment)
		address = parsed.Hostname()
	}
	if nodeType == "" {
		nodeType = strings.ToLower(strings.SplitN(link, "://", 2)[0])
	}
	if name == "" {
		name = "订阅节点"
	}
	return nodeType, address, name
}

func nodeSubscriptionDisplayName(node nodeSubscriptionItem) string {
	if node.CodeName != "" {
		return node.CodeName + "节点"
	}
	return "订阅节点"
}
