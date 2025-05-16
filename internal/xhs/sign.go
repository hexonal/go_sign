// Package xhs 提供与小红书相关的浏览器自动化与签名服务。
package xhs

import (
	"context"
	"errors"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/mxschmitt/playwright-go"
)

// Signer 封装了 Playwright 浏览器上下文和页面，用于生成小红书签名。
type Signer struct {
	pw        *playwright.Playwright
	browser   playwright.Browser
	context   playwright.BrowserContext
	page      playwright.Page
	stealthJS string
	initOnce  sync.Once
	initErr   error
}

// NewSigner 创建一个新的 Signer 实例。
// stealthJSPath 为 stealth.min.js 的文件路径。
func NewSigner(ctx context.Context, stealthJSPath string) (*Signer, error) {
	var s Signer
	var err error

	s.initOnce.Do(func() {
		s.stealthJS = stealthJSPath
		slog.Info("启动 Playwright...")
		// 启动 Playwright
		s.pw, err = playwright.Run()
		if err != nil {
			s.initErr = fmt.Errorf("启动 Playwright 失败: %w", err)
			slog.Error("Playwright 启动失败", "err", err)
			return
		}
		slog.Info("启动 Chromium...")
		s.browser, err = s.pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(true),
		})
		if err != nil {
			s.initErr = fmt.Errorf("启动 Chromium 失败: %w", err)
			slog.Error("Chromium 启动失败", "err", err)
			return
		}
		s.context, err = s.browser.NewContext()
		if err != nil {
			s.initErr = fmt.Errorf("创建浏览器上下文失败: %w", err)
			slog.Error("创建浏览器上下文失败", "err", err)
			return
		}
		// 注入 stealth.js
		if _, err := os.Stat(stealthJSPath); err != nil {
			s.initErr = fmt.Errorf("stealth.js 文件不存在: %w", err)
			slog.Error("stealth.js 文件不存在", "path", stealthJSPath, "err", err)
			return
		}
		slog.Info("注入 stealth.js", "path", stealthJSPath)
		err = s.context.AddInitScript(playwright.BrowserContextAddInitScriptOptions{
			Path: playwright.String(stealthJSPath),
		})
		if err != nil {
			s.initErr = fmt.Errorf("注入 stealth.js 失败: %w", err)
			slog.Error("注入 stealth.js 失败", "err", err)
			return
		}
		// 新建页面并访问小红书首页
		s.page, err = s.context.NewPage()
		if err != nil {
			s.initErr = fmt.Errorf("新建页面失败: %w", err)
			slog.Error("新建页面失败", "err", err)
			return
		}
		slog.Info("跳转小红书首页...")
		if _, err = s.page.Goto("https://www.xiaohongshu.com"); err != nil {
			s.initErr = fmt.Errorf("跳转小红书首页失败: %w", err)
			slog.Error("跳转小红书首页失败", "err", err)
			return
		}
		// 打印 a1 cookie
		cookies, err := s.context.Cookies()
		if err == nil {
			for _, c := range cookies {
				if c.Name == "a1" {
					slog.Info("当前浏览器 cookie 中 a1 值", "a1", c.Value)
				}
			}
		} else {
			slog.Warn("获取 cookie 失败", "err", err)
		}
	})
	if s.initErr != nil {
		return nil, s.initErr
	}
	return &s, nil
}

// SignParams 定义签名所需的参数。
type SignParams struct {
	URI        string `json:"uri"`
	Data       any    `json:"data"`
	A1         string `json:"a1"`
	WebSession string `json:"web_session"`
}

// SignResult 定义签名结果。
type SignResult struct {
	XS string `json:"x-s"`
	XT string `json:"x-t"`
}

// Sign 调用页面 JS 生成签名。
// uri: 请求路径，data: 请求数据，a1/web_session: 相关 cookie。
func (s *Signer) Sign(ctx context.Context, params SignParams) (*SignResult, error) {
	if s.page == nil {
		slog.Error("页面未初始化，无法签名")
		return nil, errors.New("页面未初始化")
	}
	slog.Info("执行签名 JS", "uri", params.URI)

	// 1. 检查 window._webmsxyw 是否存在
	exists, err := s.page.Evaluate("() => typeof window._webmsxyw === 'function'", nil)
	if err != nil {
		slog.Error("检查 window._webmsxyw 失败", "err", err)
		return nil, fmt.Errorf("检查 window._webmsxyw 失败: %w", err)
	}
	if exists != true {
		slog.Error("window._webmsxyw 未定义或未注入签名 JS")
		return nil, errors.New("window._webmsxyw 未定义或未注入签名 JS")
	}

	// 2. data 参数序列化为 JSON 字符串
	dataJSON, err := json.Marshal(params.Data)
	if err != nil {
		slog.Error("data 参数序列化失败", "err", err, "data", params.Data)
		return nil, fmt.Errorf("data 参数序列化失败: %w", err)
	}

	// 3. JS 端用 JSON.parse 还原 data
	js := `([url, dataStr]) => window._webmsxyw(url, JSON.parse(dataStr))`
	res, err := s.page.Evaluate(js, []any{params.URI, string(dataJSON)})
	if err != nil {
		slog.Error("执行签名 JS 失败", "err", err, "uri", params.URI, "data", string(dataJSON))
		return nil, fmt.Errorf("执行签名 JS 失败: %w", err)
	}
	m, ok := res.(map[string]any)
	if !ok {
		slog.Error("签名结果类型断言失败", "res", res)
		return nil, errors.New("签名结果类型断言失败")
	}
	xs, _ := m["X-s"].(string)
	xt, _ := m["X-t"].(string)
	slog.Info("签名成功", "x-s", xs, "x-t", xt, "uri", params.URI)
	return &SignResult{XS: xs, XT: xt}, nil
}

// Close 释放 Playwright 相关资源，防止资源泄漏。
// 应在服务优雅退出时调用。
func (s *Signer) Close() error {
	var firstErr error
	if s.page != nil {
		if err := s.page.Close(); err != nil {
			slog.Warn("关闭页面失败", "err", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("关闭页面失败: %w", err)
			}
		}
	}
	if s.context != nil {
		if err := s.context.Close(); err != nil {
			slog.Warn("关闭浏览器上下文失败", "err", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("关闭浏览器上下文失败: %w", err)
			}
		}
	}
	if s.browser != nil {
		if err := s.browser.Close(); err != nil {
			slog.Warn("关闭浏览器失败", "err", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("关闭浏览器失败: %w", err)
			}
		}
	}
	if s.pw != nil {
		if err := s.pw.Stop(); err != nil {
			slog.Warn("关闭 Playwright 失败", "err", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("关闭 Playwright 失败: %w", err)
			}
		}
	}
	return firstErr
}
