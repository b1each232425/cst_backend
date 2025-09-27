package ai_mark

var TestedQuestionDetails = [][]*QuestionDetail{
	{{
		ID:    101,
		Score: 8,
		Answer: `
(1) 独立式（Fat AP）组网：每个无线接入点单独配置和管理。
(2) 控制器集中式（Fit AP + AC）组网：AP 受控于无线控制器。
`,
		Rule: `
(1) 组网方式名称（2分）
    - 正确写出“独立式（Fat AP）组网”得1分。
    - 正确说明“每个无线接入点单独配置和管理”得1分。
(2) 优缺点分析（6分）
    - 独立式优点：部署简单、成本低（任意两点，每点1分，共2分）。
    - 独立式缺点：管理复杂、扩展性差、无法集中控制（任意两点，每点1分，共2分）。
    - 控制器集中式优点：集中管理、自动调优、漫游切换顺畅（任意两点，每点1分，共2分）。
    - 控制器集中式缺点：成本高、对控制器性能依赖大（任意两点，每点1分，共2分，取最高2分）。
    - 超出题意或错误描述不得分。
`,
	}},
	{{
		ID:    102,
		Score: 6,
		Answer: `
(1) OSI 七层模型：物理层、数据链路层、网络层、传输层、会话层、表示层、应用层。
(2) 应用层协议示例：HTTP、FTP、SMTP。
`,
		Rule: `
(1) 七层名称（4分）
    - 每正确写出一层名称得0.5分，最多4分。
(2) 应用层协议举例（2分）
    - 写出HTTP、FTP、SMTP中任意一个得1分，两种及以上得2分。
`,
	}},
	{{
		ID:    103,
		Score: 4,
		Answer: `
(1) 三磷酸腺苷（ATP）
(2) 主要功能：为细胞提供直接能源。
`,
		Rule: `
(1) 名称（2分）
    - 正确写出“三磷酸腺苷”或“ATP”得2分。
(2) 功能（2分）
    - 准确表述“提供细胞直接能源”或同义描述得2分。
`,
	}},
}

var TestedStudentAnswers = [][]*StudentAnswer{
	{{
		QuestionID: 101,
		Answer: `
(1) 独立式（Fat AP）组网：每个AP独立管理。
(2) 优点：部署简单，成本低。
    控制器集中式：集中管理，漫游切换顺畅，缺点成本高。
`,
	}},
	{{
		QuestionID: 102,
		Answer: `
(1) 物理层、数据链路层、网络层、传输层、会话层、表示层、应用层。
(2) HTTP。
`,
	}},
	{{
		QuestionID: 103,
		Answer: `
(1) ATP
(2) 为细胞提供能量。
`,
	}},
}

var TestedRespResults = []ResponseContent{
	{
		MarkResults: []struct {
			QuestionID int64   `json:"question_id"`
			Score      float64 `json:"score"`
			Analyze    string  `json:"analyze"`
		}{
			{
				QuestionID: 101,
				Score:      8,
				Analyze:    `答出“独立式（Fat AP）组网”得2分，独立式优点两点得2分，独立式缺点两点得2分，控制器集中式优点两点得2分，控制器集中式缺点两点得0分（未答）.`,
			},
		},
	},
	{
		MarkResults: []struct {
			QuestionID int64   `json:"question_id"`
			Score      float64 `json:"score"`
			Analyze    string  `json:"analyze"`
		}{
			{
				QuestionID: 102,
				Score:      6,
				Analyze:    `答出七层名称全部正确得4分，答出HTTP得1分，总计6分.`,
			},
		},
	},
	{
		MarkResults: []struct {
			QuestionID int64   `json:"question_id"`
			Score      float64 `json:"score"`
			Analyze    string  `json:"analyze"`
		}{
			{
				QuestionID: 103,
				Score:      4,
				Analyze:    `答出“三磷酸腺苷（ATP）”得2分，功能描述正确得2分.`,
			},
		},
	},
}
