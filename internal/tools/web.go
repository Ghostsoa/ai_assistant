package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	appconfig "ai_assistant/internal/config"
)

// ExecuteWebSearch 执行网络搜索（使用百度千帆API）
func ExecuteWebSearch(args map[string]interface{}) string {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "[✗] 搜索失败: 缺少搜索关键词"
	}

	// 检查API Key是否配置
	if appconfig.GlobalConfig.BaiduSearchKey == "" {
		return "[✗] 搜索功能未启用\n提示: 请在配置文件中添加 baidu_search_key 以启用搜索功能"
	}

	// 获取最大结果数
	maxResults := 5
	if max, ok := args["max_results"].(float64); ok {
		maxResults = int(max)
	}

	// 调用百度搜索
	results, err := searchBaidu(query, maxResults)
	if err != nil {
		return fmt.Sprintf("[✗] 搜索失败: %v", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("[搜索] 关键词 '%s' 没有找到相关结果", query)
	}

	// 格式化输出
	var output string
	output += fmt.Sprintf("[搜索] 关键词: %s\n", query)
	output += fmt.Sprintf("[搜索] 找到 %d 条结果:\n\n", len(results))

	for i, result := range results {
		output += fmt.Sprintf("%d. %s\n", i+1, result.Title)
		output += fmt.Sprintf("   URL: %s\n", result.URL)
		if result.Content != "" {
			output += fmt.Sprintf("   摘要: %s\n", result.Content)
		}
		if result.Date != "" {
			output += fmt.Sprintf("   日期: %s\n", result.Date)
		}
		output += "\n"
	}

	return output
}

// BaiduSearchResult 搜索结果
type BaiduSearchResult struct {
	Content string
	Date    string
	Title   string
	URL     string
}

// searchBaidu 调用百度千帆搜索API
func searchBaidu(query string, maxResults int) ([]BaiduSearchResult, error) {
	url := "https://qianfan.baidubce.com/v2/ai_search/web_search"

	// 构建请求体
	requestBody := map[string]interface{}{
		"messages": []map[string]string{
			{
				"content": query,
				"role":    "user",
			},
		},
		"search_source": "baidu_search_v2",
		"resource_type_filter": []map[string]interface{}{
			{
				"type":  "web",
				"top_k": maxResults,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %v", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("X-Appbuilder-Authorization", "Bearer "+appconfig.GlobalConfig.BaiduSearchKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查错误
	if code, ok := result["code"].(float64); ok && code != 0 {
		message := result["message"].(string)
		return nil, fmt.Errorf("API错误 (code:%.0f): %s", code, message)
	}

	// 提取结果
	var searchResults []BaiduSearchResult
	if references, ok := result["references"].([]interface{}); ok {
		for _, ref := range references {
			if refMap, ok := ref.(map[string]interface{}); ok {
				searchResult := BaiduSearchResult{}
				if title, ok := refMap["title"].(string); ok {
					searchResult.Title = title
				}
				if url, ok := refMap["url"].(string); ok {
					searchResult.URL = url
				}
				if content, ok := refMap["content"].(string); ok {
					searchResult.Content = content
				}
				if date, ok := refMap["date"].(string); ok {
					searchResult.Date = date
				}
				searchResults = append(searchResults, searchResult)
			}
		}
	}

	return searchResults, nil
}
