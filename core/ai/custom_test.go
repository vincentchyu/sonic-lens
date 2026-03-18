package ai

/*
{"title":"山雀","artist":"万能青年旅店","album":"冀西南林路行","lyrics":"","lang_source":"auto","lang_target":"","feedback_context":"用户对之前分析的主要反馈意见（请避免重复这些问题）：\n- 牛头不对马尾"}
*/
/*
func TestCompletions(t *testing.T) {
	ctx := context.Background()
	payload := customCompletionsRequest{
		Model: "Qwen3.5-35B-A3B-4bit",
		Prompt: buildTrackInsightMergedPrompt(
			TrackAnalysisRequest{
				Title:           "山雀",
				Artist:          "万能青年旅店",
				Album:           "冀西南林路行",
				Lyrics:          "自然赠予你\\n树冠 微风 肩头的暴雨\\n片刻后生成\\n平衡 忠诚 不息的身体\\n捕食饮水\\n清早眉间白云生\\n跳跃漫游\\n晚来拂面渤海风\\n朝霞化精灵\\n轻快 明亮 恆温的伴侣\\n她与你共存\\n违背 对抗 相同的命运\\n爱与疼痛\\n不觉茫茫道路长\\n生活历险\\n并肩莽莽原野荒\\n山崖复远望\\n仓皇 无告 不回的河流\\n平原不可见\\n晦暗 无声 未知的存亡\\n大雾重重\\n时代喧哗造物忙\\n火光忷忷\\n指引盗寇入太行\\n大雾重重\\n时代喧哗造物忙\\n火光忷忷\\n指引盗寇入太行",
				LangSource:      "auto",
				LangTarget:      "zh-CN",
				FeedbackContext: "用户对之前分析的主要反馈意见（请避免重复这些问题）：\\n- 牛头不对马尾",
			},
		),
		Stream: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(body))

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, "http://localhost:8000"+"/v1/completions", bytes.NewReader(body),
	)
	if err != nil {
		t.Fatal(err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+"6624")
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 60 * time.Minute,
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
	response := &customCompletionsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	t.Log(response)
}

func TestChatCompletions(t *testing.T) {
	ctx := context.Background()
	request := TrackAnalysisRequest{
		Title:           "山雀",
		Artist:          "万能青年旅店",
		Album:           "冀西南林路行",
		Lyrics:          "自然赠予你\\n树冠 微风 肩头的暴雨\\n片刻后生成\\n平衡 忠诚 不息的身体\\n捕食饮水\\n清早眉间白云生\\n跳跃漫游\\n晚来拂面渤海风\\n朝霞化精灵\\n轻快 明亮 恆温的伴侣\\n她与你共存\\n违背 对抗 相同的命运\\n爱与疼痛\\n不觉茫茫道路长\\n生活历险\\n并肩莽莽原野荒\\n山崖复远望\\n仓皇 无告 不回的河流\\n平原不可见\\n晦暗 无声 未知的存亡\\n大雾重重\\n时代喧哗造物忙\\n火光忷忷\\n指引盗寇入太行\\n大雾重重\\n时代喧哗造物忙\\n火光忷忷\\n指引盗寇入太行",
		LangSource:      "auto",
		LangTarget:      "zh-CN",
		FeedbackContext: "用户对之前分析的主要反馈意见（请避免重复这些问题）：\\n- 牛头不对马尾",
	}
	payload := customChatRequest{
		Model: "Qwen3.5-35B-A3B-4bit",
		Messages: []openAIChatMessage{
			{Role: "system", Content: buildTrackInsightMergedPrompt(request)},
			{Role: "user", Content: "开始你的表演"},
		},
		Stream: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(body))

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, "http://localhost:8000"+"/v1/chat/completions", bytes.NewReader(body),
	)
	if err != nil {
		t.Fatal(err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+"6624")
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 60 * time.Minute,
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
	response := &customChatResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	t.Log(response)
}*/
