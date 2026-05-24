const zh = {
  common: {
    apply: '应用',
    clear: '清空',
    creating: '创建中...',
    required: '必填',
    sending: '发送中...',
    tryAgain: '请重试'
  },
  auth: {
    oauth: {
      openaiCompleting: '正在完成 OpenAI 授权...',
      openaiCompleteSuccess: 'OpenAI 授权完成',
      openaiCompleteFailed: 'OpenAI 授权失败'
    }
  },
  keys: {
    status: {
      active: '启用',
      inactive: '禁用',
      quota_exhausted: '额度耗尽',
      expired: '已过期'
    }
  },
  admin: {
    users: {
      passwordCopied: '密码已复制'
    },
    groups: {
      failedToSave: '保存分组失败',
      platforms: {
        all: '全部平台',
        anthropic: 'Anthropic',
        openai: 'OpenAI',
        gemini: 'Gemini',
        antigravity: 'Antigravity',
        claude: 'Claude',
        openrouter: 'OpenRouter',
        vertex: 'Vertex',
        bedrock: 'AWS Bedrock'
      }
    },
    redeem: {
      types: {
        balance: '余额',
        concurrency: '并发',
        subscription: '订阅'
      }
    },
    accounts: {
      searchIPs: '搜索出口 IP...',
      fromModel: '来源模型',
      toModel: '目标模型',
      crsProxyName: '代理',
      crsProxyStatusMatched: '代理已匹配',
      crsProxyStatusNotFound: '代理未找到',
      crsProxyStatusConflict: '代理冲突',
      crsProxyStatusMissing: '未配置代理',
      crsWarnings: '同步警告',
      bulkTestActivateSummary: '批量测试完成：成功 {success} 个，失败 {failed} 个，激活 {activated} 个，停用 {deactivated} 个',
      bulkTestActivateTimeout: '批量测试激活超时，请稍后刷新查看结果',
      bulkActions: {
        testActivate: '批量测试激活',
        testActivating: '批量测试激活中...',
        testActivateConfirm: '确定要批量测试激活选中的 {count} 个账号吗？'
      },
      oauth: {
        openAuthUrl: '打开授权链接',
        openai: {
          mobileRefreshTokenAuth: '手动输入 Mobile RT',
          accessTokenAuth: '手动输入 AT'
        }
      }
    },
    channels: {
      noGroupsSelected: '{platform} 平台未选择分组，请至少选择一个分组或禁用该平台',
      emptyModelsInPricing: '请为 {platform} 平台下的定价条目添加模型，或删除空定价条目'
    },
    ops: {
      result: '结果',
      timeRange: {
        custom: '自定义'
      },
      customTimeRange: {
        startTime: '开始时间',
        endTime: '结束时间'
      },
      runtime: {
        metricThresholds: '指标阈值',
        metricThresholdsHint: '配置运维健康评分和告警诊断使用的阈值',
        upstreamErrorRateMaxPercent: '上游错误率上限（%）',
        upstreamErrorRateMaxPercentHint: '超过该值时提示上游错误率异常',
        requestErrorRateMaxPercent: '请求错误率上限（%）',
        requestErrorRateMaxPercentHint: '超过该值时提示请求错误率异常',
        slaMinPercent: 'SLA 下限（%）',
        slaMinPercentHint: '低于该值时提示 SLA 风险',
        ttftP99MaxMs: 'P99 首 Token 延迟上限（毫秒）',
        ttftP99MaxMsHint: '超过该值时提示首 Token 延迟异常'
      }
    },
    settings: {
      telegram: {
        title: 'Telegram 通知',
        description: '配置 Telegram Bot，用于发送系统通知和告警。',
        enabled: '启用 Telegram 通知',
        enabledHint: '开启后系统会通过 Telegram 发送相关通知。',
        botToken: 'Bot Token',
        botTokenPlaceholder: '请输入 Telegram Bot Token',
        botTokenConfiguredPlaceholder: 'Bot Token 已配置，留空以保留当前值。',
        botTokenHint: '填写后会覆盖当前 Telegram Bot Token。',
        botTokenConfiguredHint: 'Bot Token 已配置，留空以保留当前值。',
        chatIds: 'Chat ID 列表',
        chatIdsPlaceholder: '每行一个 Chat ID，例如 123456789',
        chatIdsHint: '支持多个 Chat ID，每行一个或用逗号分隔。',
        proxyUrls: '代理 URL 列表',
        proxyUrlsPlaceholder: '每行一个代理地址，例如 http://host.docker.internal:65182',
        proxyUrlsHint: '可选。用于 Telegram API 出口代理，支持多个地址。'
      },
      testTelegram: {
        sendTestTelegram: '发送测试 Telegram',
        sending: '发送中...',
        enterChatIdsHint: '请输入 Telegram Chat ID'
      },
      testTelegramSent: '测试 Telegram 已发送',
      failedToSendTestTelegram: '发送测试 Telegram 失败'
    },
    errorPassthrough: {
      matchMode: {
        any: '状态码或关键词',
        all: '状态码且关键词',
        anyHint: '状态码匹配任一错误码，或消息包含任一关键词',
        allHint: '状态码匹配错误码，且消息包含关键词'
      }
    }
  },
  payment: {
    admin: {
      paymentDistribution: '支付方式分布',
      dailyRevenue: '每日收入',
      noData: '暂无数据',
      revenue: '收入',
      orderCount: '订单数'
    }
  }
}

const en = {
  common: {
    apply: 'Apply',
    clear: 'Clear',
    creating: 'Creating...',
    required: 'required',
    sending: 'Sending...',
    tryAgain: 'Please try again'
  },
  auth: {
    oauth: {
      openaiCompleting: 'Completing OpenAI authorization...',
      openaiCompleteSuccess: 'OpenAI authorization completed',
      openaiCompleteFailed: 'OpenAI authorization failed'
    }
  },
  keys: {
    status: {
      active: 'Active',
      inactive: 'Inactive',
      quota_exhausted: 'Quota exhausted',
      expired: 'Expired'
    }
  },
  admin: {
    users: {
      passwordCopied: 'Password copied'
    },
    groups: {
      failedToSave: 'Failed to save group',
      platforms: {
        all: 'All Platforms',
        anthropic: 'Anthropic',
        openai: 'OpenAI',
        gemini: 'Gemini',
        antigravity: 'Antigravity',
        claude: 'Claude',
        openrouter: 'OpenRouter',
        vertex: 'Vertex',
        bedrock: 'AWS Bedrock'
      }
    },
    redeem: {
      types: {
        balance: 'Balance',
        concurrency: 'Concurrency',
        subscription: 'Subscription'
      }
    },
    accounts: {
      searchIPs: 'Search outbound IPs...',
      fromModel: 'From model',
      toModel: 'To model',
      crsProxyName: 'Proxy',
      crsProxyStatusMatched: 'Proxy matched',
      crsProxyStatusNotFound: 'Proxy not found',
      crsProxyStatusConflict: 'Proxy conflict',
      crsProxyStatusMissing: 'Proxy missing',
      crsWarnings: 'Sync warnings',
      bulkTestActivateSummary: 'Bulk test completed: {success} succeeded, {failed} failed, {activated} activated, {deactivated} deactivated',
      bulkTestActivateTimeout: 'Bulk activation test timed out. Refresh later to view results.',
      bulkActions: {
        testActivate: 'Test Activate',
        testActivating: 'Testing activation...',
        testActivateConfirm: 'Test activation for the selected {count} account(s)?',
        trueRefreshToken: 'True Refresh Token',
        trueRefreshTokenStarted: 'True refresh token job started'
      },
      oauth: {
        openAuthUrl: 'Open authorization URL',
        openai: {
          mobileRefreshTokenAuth: 'Enter Mobile RT manually',
          accessTokenAuth: 'Enter AT manually'
        }
      }
    },
    channels: {
      noGroupsSelected: '{platform} has no groups selected. Select at least one group or disable this platform.',
      emptyModelsInPricing: 'Pricing entries under {platform} have no models. Add models or remove the empty entries.'
    },
    ops: {
      result: 'Result',
      timeRange: {
        custom: 'Custom'
      },
      customTimeRange: {
        startTime: 'Start time',
        endTime: 'End time'
      },
      runtime: {
        metricThresholds: 'Metric Thresholds',
        metricThresholdsHint: 'Configure thresholds used by ops health scoring and diagnostics',
        upstreamErrorRateMaxPercent: 'Max upstream error rate (%)',
        upstreamErrorRateMaxPercentHint: 'Warn when upstream error rate exceeds this value',
        requestErrorRateMaxPercent: 'Max request error rate (%)',
        requestErrorRateMaxPercentHint: 'Warn when request error rate exceeds this value',
        slaMinPercent: 'Minimum SLA (%)',
        slaMinPercentHint: 'Warn when SLA falls below this value',
        ttftP99MaxMs: 'P99 TTFT max (ms)',
        ttftP99MaxMsHint: 'Warn when P99 time-to-first-token exceeds this value'
      }
    },
    settings: {
      telegram: {
        title: 'Telegram Notifications',
        description: 'Configure a Telegram Bot for system notifications and alerts.',
        enabled: 'Enable Telegram Notifications',
        enabledHint: 'When enabled, the system sends related notifications through Telegram.',
        botToken: 'Bot Token',
        botTokenPlaceholder: 'Enter Telegram Bot Token',
        botTokenConfiguredPlaceholder: 'Bot Token configured. Leave empty to keep the current value.',
        botTokenHint: 'Entering a value will replace the current Telegram Bot Token.',
        botTokenConfiguredHint: 'Bot Token configured. Leave empty to keep the current value.',
        chatIds: 'Chat IDs',
        chatIdsPlaceholder: 'One Chat ID per line, e.g. 123456789',
        chatIdsHint: 'Multiple Chat IDs are supported, one per line or comma-separated.',
        proxyUrls: 'Proxy URLs',
        proxyUrlsPlaceholder: 'One proxy URL per line, e.g. http://host.docker.internal:65182',
        proxyUrlsHint: 'Optional. Telegram API outbound proxies; multiple URLs are supported.'
      },
      testTelegram: {
        sendTestTelegram: 'Send Test Telegram',
        sending: 'Sending...',
        enterChatIdsHint: 'Please enter Telegram Chat ID'
      },
      testTelegramSent: 'Test Telegram sent successfully',
      failedToSendTestTelegram: 'Failed to send test Telegram'
    },
    errorPassthrough: {
      matchMode: {
        any: 'Code OR Keyword',
        all: 'Code AND Keyword',
        anyHint: 'Status code matches any error code, OR message contains any keyword',
        allHint: 'Status code matches an error code, AND message contains a keyword'
      }
    }
  },
  payment: {
    admin: {
      paymentDistribution: 'Payment Distribution',
      dailyRevenue: 'Daily Revenue',
      noData: 'No data',
      revenue: 'Revenue',
      orderCount: 'Orders'
    }
  }
}

export const supplementalLocales = { zh, en } as const
