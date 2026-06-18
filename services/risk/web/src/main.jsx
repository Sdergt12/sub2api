import React, { useEffect, useMemo, useState } from "react";
import ReactDOM from "react-dom/client";
import "./styles.css";

const searchParams = new URLSearchParams(window.location.search);
const embedToken = searchParams.get("k") || "";
const forceEmbed = searchParams.get("embed") === "1";
const defaultTab = normalizeTab(searchParams.get("tab"));
const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL || "").replace(/\/$/, "");

function normalizeTab(value) {
  return ["operations", "overview", "users", "events"].includes(value) ? value : "operations";
}

function getApiBaseUrl() {
  return apiBaseUrl || "/api/risk";
}

function buildApiUrl(path, params) {
  const query = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") {
      return;
    }
    query.set(key, String(value));
  });

  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  const queryText = query.toString();

  return queryText
    ? `${getApiBaseUrl()}${normalizedPath}?${queryText}`
    : `${getApiBaseUrl()}${normalizedPath}`;
}

function buildHeaders(extraHeaders = {}) {
  const headers = {
    Accept: "application/json",
    ...extraHeaders,
  };

  // 统一通过嵌入 token 控制读取权限，避免外链页面被直接抓取。
  if (embedToken) {
    headers["X-Risk-Embed-Token"] = embedToken;
  }

  return headers;
}

async function fetchJson(path, params, options = {}) {
  const response = await fetch(buildApiUrl(path, params), {
    method: options.method || "GET",
    headers: buildHeaders(options.headers),
  });

  if (!response.ok) {
    let message = `HTTP ${response.status}`;
    try {
      const payload = await response.json();
      if (payload?.error) {
        message = payload.error;
      }
    } catch {
      // ignore parsing error and keep fallback message
    }
    throw new Error(message);
  }

  return response.json();
}

function formatDateTime(value) {
  if (!value) {
    return "-";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString("zh-CN", { hour12: false });
}

function formatShortDate(value) {
  if (!value) {
    return "-";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleDateString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
  });
}

function formatRelativeTime(value) {
  if (!value) {
    return "未刷新";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  const deltaSeconds = Math.round((Date.now() - date.getTime()) / 1000);
  const abs = Math.abs(deltaSeconds);

  if (abs < 60) {
    return deltaSeconds >= 0 ? `${abs} 秒前` : `${abs} 秒后`;
  }
  if (abs < 3600) {
    const minutes = Math.round(abs / 60);
    return deltaSeconds >= 0 ? `${minutes} 分钟前` : `${minutes} 分钟后`;
  }
  if (abs < 86400) {
    const hours = Math.round(abs / 3600);
    return deltaSeconds >= 0 ? `${hours} 小时前` : `${hours} 小时后`;
  }

  const days = Math.round(abs / 86400);
  return deltaSeconds >= 0 ? `${days} 天前` : `${days} 天后`;
}

function formatNumber(value) {
  const safe = Number(value || 0);
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 }).format(safe);
}

function formatDecimal(value, digits = 2) {
  const safe = Number(value || 0);
  return new Intl.NumberFormat("en-US", {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(safe);
}

function formatMoney(value) {
  const safe = Number(value || 0);
  return `$${formatDecimal(safe, 2)}`;
}

function formatSignedMoney(value) {
  const safe = Number(value || 0);
  if (safe > 0) return `+${formatMoney(safe)}`;
  if (safe < 0) return `-${formatMoney(Math.abs(safe))}`;
  return formatMoney(0);
}

function formatCny(value) {
  const safe = Number(value || 0);
  return `¥${formatDecimal(safe, 2)}`;
}

function formatPercent(value) {
  const safe = Number(value || 0);
  return `${(safe * 100).toFixed(1)}%`;
}

function formatMs(value) {
  const safe = Number(value || 0);
  return `${Math.round(safe)} ms`;
}

function riskLevelTone(level) {
  switch (level) {
    case "high":
      return "danger";
    case "warning":
      return "warning";
    case "normal":
      return "success";
    default:
      return "neutral";
  }
}

function severityTone(level) {
  switch (level) {
    case "high":
      return "danger";
    case "warning":
      return "warning";
    case "info":
      return "info";
    default:
      return "neutral";
  }
}

function eventStatusTone(status) {
  switch (status) {
    case "open":
      return "danger";
    case "acknowledged":
      return "warning";
    case "resolved":
      return "success";
    case "ignored":
      return "neutral";
    default:
      return "neutral";
  }
}

function freshnessTone(value) {
  if (!value) {
    return "warning";
  }

  const minutes = (Date.now() - new Date(value).getTime()) / 60000;
  if (minutes <= 15) {
    return "success";
  }
  if (minutes <= 60) {
    return "warning";
  }
  return "danger";
}

function renderEventDetail(detail) {
  if (!detail || typeof detail !== "object") {
    return [];
  }
  return Object.entries(detail);
}

function SectionState({ loading, error, empty, emptyText, children }) {
  if (loading) {
    return <div className="state-box">正在加载数据...</div>;
  }

  if (error) {
    return <div className="state-box state-box-danger">{error}</div>;
  }

  if (empty) {
    return <div className="state-box">{emptyText}</div>;
  }

  return children;
}

function Pill({ tone = "neutral", children }) {
  return <span className={`pill pill-${tone}`}>{children}</span>;
}

function StatTile({ label, value, hint, tone = "neutral" }) {
  return (
    <div className={`stat-tile stat-tile-${tone}`}>
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
      {hint ? <div className="stat-hint">{hint}</div> : null}
    </div>
  );
}

function MetricRow({ label, value, hint, tone = "neutral" }) {
  return (
    <div className={`metric-row metric-row-${tone}`}>
      <div>
        <div className="metric-label">{label}</div>
        {hint ? <div className="metric-hint">{hint}</div> : null}
      </div>
      <div className="metric-value">{value}</div>
    </div>
  );
}

function ProgressBar({ value, max, tone = "neutral" }) {
  const safeValue = Number(value || 0);
  const safeMax = Math.max(1, Number(max || 0));
  const ratio = Math.min(100, Math.max(0, (safeValue / safeMax) * 100));

  return (
    <div className="progress-track">
      <div className={`progress-fill progress-fill-${tone}`} style={{ width: `${ratio}%` }} />
    </div>
  );
}

function SectionHeader({ eyebrow, title, description, actions }) {
  return (
    <div className="section-header">
      <div>
        <div className="section-eyebrow">{eyebrow}</div>
        <h2 className="section-title">{title}</h2>
        <p className="section-description">{description}</p>
      </div>
      {actions ? <div className="section-actions">{actions}</div> : null}
    </div>
  );
}

function FilterField({ label, children }) {
  return (
    <label className="filter-field">
      <span className="filter-label">{label}</span>
      {children}
    </label>
  );
}

function ToolbarButton({ primary = false, children, ...props }) {
  return (
    <button
      type="button"
      className={`toolbar-button ${primary ? "toolbar-button-primary" : ""}`}
      {...props}
    >
      {children}
    </button>
  );
}

function TabButton({ active, children, onClick }) {
  return (
    <button type="button" className={`tab-button ${active ? "tab-button-active" : ""}`} onClick={onClick}>
      {children}
    </button>
  );
}

function SignalMeter({ items }) {
  const total = Math.max(1, items.reduce((accumulator, item) => accumulator + Number(item.value || 0), 0));

  return (
    <div className="signal-meter">
      {items.map((item) => (
        <div key={item.label} className="signal-meter-row">
          <div className="signal-meter-head">
            <span>{item.label}</span>
            <span>{formatNumber(item.value)}</span>
          </div>
          <ProgressBar value={item.value} max={total} tone={item.tone} />
          <div className="signal-meter-foot">{item.description}</div>
        </div>
      ))}
    </div>
  );
}

function DailyTrendChart({ rows }) {
  const maxRequests = Math.max(1, ...rows.map((item) => item.usage_requests || 0));
  const maxSigns = Math.max(1, ...rows.map((item) => item.sign_users || 0));
  const maxPaid = Math.max(1, ...rows.map((item) => item.paid_orders || 0));

  return (
    <div className="trend-chart">
      {rows.map((item) => (
        <div key={item.date} className="trend-day-card">
          <div className="trend-day-head">
            <div className="trend-day-date">{formatShortDate(item.date)}</div>
            <div className="trend-day-meta">{formatNumber(item.usage_active_users || 0)} 活跃</div>
          </div>
          <div className="trend-bars">
            <div className="trend-bar-item">
              <div className="trend-bar-label">签到</div>
              <ProgressBar value={item.sign_users || 0} max={maxSigns} tone="success" />
              <div className="trend-bar-value">{formatNumber(item.sign_users || 0)}</div>
            </div>
            <div className="trend-bar-item">
              <div className="trend-bar-label">支付</div>
              <ProgressBar value={item.paid_orders || 0} max={maxPaid} tone="warning" />
              <div className="trend-bar-value">{formatNumber(item.paid_orders || 0)}</div>
            </div>
            <div className="trend-bar-item">
              <div className="trend-bar-label">请求</div>
              <ProgressBar value={item.usage_requests || 0} max={maxRequests} tone="info" />
              <div className="trend-bar-value">{formatNumber(item.usage_requests || 0)}</div>
            </div>
          </div>
          <div className="trend-day-footer">
            <span>{formatMoney(item.sign_reward_total || 0)} 奖励</span>
            <span>{formatMoney(item.paid_amount || 0)} 充值</span>
          </div>
        </div>
      ))}
    </div>
  );
}

function RankingList({ title, description, items, metricFormatter, secondaryLabel }) {
  return (
    <div className="panel-card">
      <div className="panel-card-head">
        <div className="panel-card-title">{title}</div>
        <div className="panel-card-subtitle">{description}</div>
      </div>
      <div className="ranking-list">
        {items.length === 0 ? (
          <div className="empty-inline">暂无数据</div>
        ) : (
          items.map((item, index) => (
            <div key={`${title}-${item.user_id}-${index}`} className="ranking-row">
              <div className="ranking-left">
                <div className="ranking-index">{index + 1}</div>
                <div>
                  <div className="ranking-user">{item.user_label || item.user_id}</div>
                  <div className="ranking-subtitle">UID {item.user_id} · {secondaryLabel(item)}</div>
                </div>
              </div>
              <div className="ranking-value">{metricFormatter(item.metric_value)}</div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function ConversionFunnel({ conversion, payment, sign }) {
  const signedUsers = Number(conversion?.signed_users_24h || 0);
  const paidUsers = Number(conversion?.paid_users_24h || 0);
  const overlapUsers = Number(conversion?.signed_then_paid_users_24h || 0);
  const createdOrders = Number(payment?.today_created_orders || 0);
  const paidOrders = Number(payment?.today_paid_orders || 0);
  const funnelMax = Math.max(1, signedUsers, paidUsers, overlapUsers, createdOrders, paidOrders);

  const steps = [
    {
      label: "24h 签到用户",
      value: signedUsers,
      tone: "success",
      hint: `今日成功签到 ${formatNumber(sign?.today_success_users || 0)} 人`,
    },
    {
      label: "24h 支付用户",
      value: paidUsers,
      tone: "warning",
      hint: "完成支付的独立用户",
    },
    {
      label: "签到后付费",
      value: overlapUsers,
      tone: "info",
      hint: `转化率 ${formatPercent(conversion?.signed_to_paid_rate_24h || 0)}`,
    },
    {
      label: "今日已创建订单",
      value: createdOrders,
      tone: "neutral",
      hint: "当日发起充值的订单数",
    },
    {
      label: "今日已支付订单",
      value: paidOrders,
      tone: "warning",
      hint: `支付金额 ${formatMoney(payment?.today_paid_amount || 0)}`,
    },
  ];

  return (
    <div className="panel-card">
      <div className="panel-card-head">
        <div className="panel-card-title">转化链路</div>
        <div className="panel-card-subtitle">把签到、下单、支付放到同一视角看，不再只看单点数字。</div>
      </div>
      <div className="funnel-list">
        {steps.map((item) => (
          <div key={item.label} className="funnel-row">
            <div className="funnel-copy">
              <div className="funnel-label">{item.label}</div>
              <div className="funnel-hint">{item.hint}</div>
            </div>
            <div className="funnel-visual">
              <ProgressBar value={item.value} max={funnelMax} tone={item.tone} />
            </div>
            <div className="funnel-value">{formatNumber(item.value)}</div>
          </div>
        ))}
      </div>
    </div>
  );
}

function InsightPanel({ overview }) {
  const sign = overview?.sign;
  const payment = overview?.payment;
  const usage = overview?.usage;
  const conversion = overview?.conversion;

  const rewardPerSignedUser = Number(sign?.today_success_users || 0)
    ? Number(sign?.today_total_reward || 0) / Number(sign?.today_success_users || 0)
    : 0;
  const paidOrderRate = Number(payment?.today_created_orders || 0)
    ? Number(payment?.today_paid_orders || 0) / Number(payment?.today_created_orders || 0)
    : 0;
  const avgPaidUserTopup = Number(payment?.today_paid_users || 0)
    ? Number(payment?.today_paid_amount || 0) / Number(payment?.today_paid_users || 0)
    : 0;

  return (
    <div className="insight-grid">
      <div className="insight-card">
        <div className="insight-title">签到成本</div>
        <div className="insight-value">{formatMoney(rewardPerSignedUser)}</div>
        <div className="insight-copy">按今天已签到用户平均发放，便于判断奖励是否过高。</div>
      </div>
      <div className="insight-card">
        <div className="insight-title">订单支付率</div>
        <div className="insight-value">{formatPercent(paidOrderRate)}</div>
        <div className="insight-copy">从已创建订单到成功支付的完成率，低了就要先查支付链路。</div>
      </div>
      <div className="insight-card">
        <div className="insight-title">付费用户客单</div>
        <div className="insight-value">{formatMoney(avgPaidUserTopup)}</div>
        <div className="insight-copy">今天已付费用户的人均充值额，用来看充值意愿是否真实。</div>
      </div>
      <div className="insight-card">
        <div className="insight-title">请求体验</div>
        <div className="insight-value">{formatMs(usage?.p90_latency_ms || 0)}</div>
        <div className="insight-copy">这里看 P90 总耗时。高峰明显升高时，优先查节点和模型路由。</div>
      </div>
      <div className="insight-card">
        <div className="insight-title">签到后付费</div>
        <div className="insight-value">{formatPercent(conversion?.signed_to_paid_rate_24h || 0)}</div>
        <div className="insight-copy">衡量签到到底是在补贴留存，还是已经能推升充值转化。</div>
      </div>
    </div>
  );
}

function OverviewPage({ overview, loading, error, onRefresh }) {
  const warningUsers = Number(overview?.warning_users || 0);
  const highRiskUsers = Number(overview?.high_risk_users || 0);
  const openEvents = Number(overview?.events_unprocessed || 0);
  const staleUsers = Number(overview?.stale_users || 0);

  const signalRows = [
    {
      label: "待处理事件",
      value: openEvents,
      tone: openEvents > 0 ? "danger" : "success",
      description: "还在 open 状态的风险事件，应该优先处理。",
    },
    {
      label: "高风险用户",
      value: highRiskUsers,
      tone: highRiskUsers > 0 ? "danger" : "success",
      description: "当前窗口内仍处于高风险等级的活跃用户。",
    },
    {
      label: "预警用户",
      value: warningUsers,
      tone: warningUsers > 0 ? "warning" : "success",
      description: "存在异常趋势，但还没进入高风险级别。",
    },
    {
      label: "陈旧快照",
      value: staleUsers,
      tone: "neutral",
      description: "历史命中但已不在当前窗口活跃的快照，不应混入实时判断。",
    },
  ];

  return (
    <section className="page-section">
      <SectionHeader
        eyebrow="Risk workspace"
        title="风控总览"
        description="这一页只回答三个问题：现在有没有问题、问题主要在哪、这些数据是不是新鲜的。"
        actions={<ToolbarButton onClick={onRefresh}>刷新概览</ToolbarButton>}
      />

      <SectionState loading={loading} error={error} empty={!overview} emptyText="暂无风控概览数据">
        <div className="hero-panel">
          <div className="hero-copy">
            <div className="hero-badge-row">
              <Pill tone={freshnessTone(overview?.last_refresh_at)}>{formatRelativeTime(overview?.last_refresh_at)}</Pill>
              <Pill tone="neutral">
                窗口 {formatDateTime(overview?.window_start)} - {formatDateTime(overview?.window_end)}
              </Pill>
            </div>
            <div className="hero-title">当前待处理风险事件 {formatNumber(openEvents)}</div>
            <div className="hero-description">
              风险中心第一屏不再堆表格。先看 open 事件，再看高风险和预警用户，最后确认是不是被陈旧快照污染。
            </div>
            <div className="hero-stat-strip">
              <StatTile
                label="高风险用户"
                value={formatNumber(highRiskUsers)}
                hint="需要优先复核"
                tone={highRiskUsers > 0 ? "danger" : "success"}
              />
              <StatTile
                label="预警用户"
                value={formatNumber(warningUsers)}
                hint="异常趋势待观察"
                tone={warningUsers > 0 ? "warning" : "success"}
              />
              <StatTile
                label="陈旧快照"
                value={formatNumber(staleUsers)}
                hint="历史痕迹，不能直接当实时风险"
                tone="neutral"
              />
            </div>
          </div>

          <div className="hero-board">
            <div className="panel-card-head">
              <div className="panel-card-title">信号分布</div>
              <div className="panel-card-subtitle">用比例快速判断当前问题主要来自哪里。</div>
            </div>
            <SignalMeter items={signalRows} />
          </div>
        </div>

        <div className="content-grid content-grid-2">
          <div className="panel-card">
            <div className="panel-card-head">
              <div className="panel-card-title">数据新鲜度</div>
              <div className="panel-card-subtitle">先判断数据可靠，再决定是否处理。</div>
            </div>
            <MetricRow
              label="最近刷新"
              value={formatDateTime(overview?.last_refresh_at)}
              hint={formatRelativeTime(overview?.last_refresh_at)}
              tone={freshnessTone(overview?.last_refresh_at)}
            />
            <MetricRow
              label="今日新增事件"
              value={formatNumber(overview?.events_today || 0)}
              hint="按发生时间统计"
              tone="info"
            />
          </div>

          <div className="panel-card">
            <div className="panel-card-head">
              <div className="panel-card-title">处理顺序</div>
              <div className="panel-card-subtitle">把判断步骤写清楚，避免不同人看同一面板得出不同结论。</div>
            </div>
            <ul className="signal-list">
              <li>先处理 open 事件，确认是否仍在持续发生。</li>
              <li>再看高风险和预警用户是否仍处于 active 状态。</li>
              <li>陈旧快照只作为历史痕迹，不参与当下封控判断。</li>
            </ul>
          </div>
        </div>
      </SectionState>
    </section>
  );
}

function OperationsPage({ overview, loading, error, onRefresh }) {
  const sign = overview?.sign;
  const payment = overview?.payment;
  const usage = overview?.usage;
  const trends = overview?.trends || [];
  const topPayers = overview?.top_payers || [];
  const topCost = overview?.top_cost || [];

  return (
    <section className="page-section">
      <SectionHeader
        eyebrow="Operations"
        title="运营统计"
        description="把签到、充值、转化、请求体验放到同一个页面里，避免只看一个数字就下结论。"
        actions={<ToolbarButton onClick={onRefresh}>刷新统计</ToolbarButton>}
      />

      <SectionState loading={loading} error={error} empty={!overview} emptyText="暂无运营统计数据">
        <div className="summary-hero">
          <div className="summary-main">
            <div className="summary-main-label">今日发放与回收</div>
            <div className="summary-main-value">{formatMoney(sign?.today_total_reward || 0)}</div>
            <div className="summary-main-copy">
              今日签到 {formatNumber(sign?.today_success_users || 0)} 人，平均奖励 {formatMoney(sign?.today_avg_reward || 0)}。
            </div>
          </div>
          <div className="summary-side-grid">
            <StatTile
              label="今日支付订单"
              value={formatNumber(payment?.today_paid_orders || 0)}
              hint={`支付用户 ${formatNumber(payment?.today_paid_users || 0)}`}
              tone="warning"
            />
            <StatTile
              label="今日充值金额"
              value={formatMoney(payment?.today_paid_amount || 0)}
              hint={`客单价 ${formatMoney(payment?.avg_paid_order_amount || 0)}`}
              tone="info"
            />
            <StatTile
              label="24h 请求量"
              value={formatNumber(usage?.requests_24h || 0)}
              hint={`活跃用户 ${formatNumber(usage?.active_users_24h || 0)}`}
              tone="neutral"
            />
            <StatTile
              label="P90 延迟"
              value={formatMs(usage?.p90_latency_ms || 0)}
              hint={`首包 P90 ${formatMs(usage?.p90_first_token_ms || 0)}`}
              tone="danger"
            />
          </div>
        </div>

        <ConversionFunnel conversion={overview?.conversion} payment={payment} sign={sign} />

        <InsightPanel overview={overview} />

        <div className="panel-card">
          <div className="panel-card-head">
            <div className="panel-card-title">近 7 天趋势</div>
            <div className="panel-card-subtitle">把签到、充值、请求量做成同一时间轴，方便看奖励调整后的反馈。</div>
          </div>
          <DailyTrendChart rows={trends} />
        </div>

        <div className="content-grid content-grid-2">
          <RankingList
            title="近 7 天充值排行"
            description="看真实付费用户是谁，避免只盯着签到流量。"
            items={topPayers}
            metricFormatter={formatMoney}
            secondaryLabel={(item) => `${formatNumber(item.paid_orders || 0)} 笔订单`}
          />
          <RankingList
            title="近 24 小时成本排行"
            description="看谁在消耗额度和节点资源，方便和风险页联动判断。"
            items={topCost}
            metricFormatter={formatMoney}
            secondaryLabel={(item) => `${formatNumber(item.request_count || 0)} 次请求`}
          />
        </div>
      </SectionState>
    </section>
  );
}

function UsersTable({ items }) {
  return (
    <div className="table-shell">
      <table className="data-table">
        <thead>
          <tr>
            <th>用户</th>
            <th>等级</th>
            <th>分值</th>
            <th>请求</th>
            <th>成本</th>
            <th>IP</th>
            <th>状态</th>
            <th>模型</th>
            <th>最后事件</th>
            <th>最近刷新</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={`${item.user_id}-${item.id}`}>
              <td>
                <div className="table-primary">{item.user_id}</div>
              </td>
              <td><Pill tone={riskLevelTone(item.risk_level)}>{item.risk_level || "-"}</Pill></td>
              <td>{formatNumber(item.risk_score)}</td>
              <td>
                <div className="table-primary">{formatNumber(item.request_count_1h)} / {formatNumber(item.request_count_24h)}</div>
                <div className="table-secondary">1h / 24h</div>
              </td>
              <td>
                <div className="table-primary">{formatMoney(item.cost_24h)}</div>
                <div className="table-secondary">1h {formatMoney(item.cost_1h)}</div>
              </td>
              <td>{formatNumber(item.unique_ip_24h)}</td>
              <td><Pill tone={item.is_stale ? "neutral" : "success"}>{item.is_stale ? "stale" : "active"}</Pill></td>
              <td className="table-wrap">{(item.top_models || []).join(", ") || "-"}</td>
              <td>{formatDateTime(item.last_event_at)}</td>
              <td>{formatDateTime(item.refreshed_at)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function UsersPage({
  items,
  total,
  page,
  pageSize,
  filters,
  loading,
  error,
  onRefresh,
  onPageChange,
  onFilterChange,
}) {
  const visibleSummary = useMemo(
    () =>
      items.reduce(
        (accumulator, item) => {
          accumulator.total += 1;
          if (item.is_stale) {
            accumulator.stale += 1;
          }
          if (item.risk_level === "high") {
            accumulator.high += 1;
          } else if (item.risk_level === "warning") {
            accumulator.warning += 1;
          } else {
            accumulator.normal += 1;
          }
          return accumulator;
        },
        { total: 0, high: 0, warning: 0, normal: 0, stale: 0 }
      ),
    [items]
  );

  return (
    <section className="page-section">
      <SectionHeader
        eyebrow="Users"
        title="风险用户"
        description="用户页按“当前是否活跃”和“风险等级”分读。默认隐藏 stale，避免历史快照污染判断。"
        actions={<ToolbarButton onClick={onRefresh}>刷新用户</ToolbarButton>}
      />

      <div className="filter-grid">
        <FilterField label="用户 ID">
          <input value={filters.user_id} onChange={(event) => onFilterChange("user_id", event.target.value)} placeholder="搜索用户 ID" />
        </FilterField>
        <FilterField label="风险等级">
          <select value={filters.risk_level} onChange={(event) => onFilterChange("risk_level", event.target.value)}>
            <option value="">全部等级</option>
            <option value="high">high</option>
            <option value="warning">warning</option>
            <option value="normal">normal</option>
          </select>
        </FilterField>
        <label className="toggle-field">
          <input
            type="checkbox"
            checked={filters.include_stale}
            onChange={(event) => onFilterChange("include_stale", event.target.checked)}
          />
          <span>包含 stale 快照</span>
        </label>
      </div>

      <div className="kpi-grid compact-grid">
        <StatTile label="当前页用户" value={formatNumber(visibleSummary.total)} hint={`总量 ${formatNumber(total)}`} tone="neutral" />
        <StatTile label="高风险" value={formatNumber(visibleSummary.high)} hint="当前页" tone={visibleSummary.high > 0 ? "danger" : "success"} />
        <StatTile label="预警" value={formatNumber(visibleSummary.warning)} hint="当前页" tone={visibleSummary.warning > 0 ? "warning" : "success"} />
        <StatTile label="陈旧快照" value={formatNumber(visibleSummary.stale)} hint="当前页" tone="neutral" />
      </div>

      <SectionState loading={loading} error={error} empty={!loading && !error && items.length === 0} emptyText="没有命中的风险用户">
        <>
          <UsersTable items={items} />
          <PaginationBar page={page} pageSize={pageSize} total={total} onPageChange={onPageChange} />
        </>
      </SectionState>
    </section>
  );
}

function EventCard({ item }) {
  return (
    <div className="event-card">
      <div className="event-card-top">
        <div>
          <div className="event-title">{item.title || "-"}</div>
          <div className="event-subtitle">
            {item.event_type || "-"} · 用户 {item.user_id || "-"} · 指纹 {item.event_fingerprint || "-"}
          </div>
        </div>
        <div className="event-badges">
          <Pill tone={eventStatusTone(item.status)}>{item.status || "-"}</Pill>
          <Pill tone={severityTone(item.severity)}>{item.severity || "-"}</Pill>
        </div>
      </div>

      <div className="event-metrics-grid">
        <StatTile label="命中次数" value={formatNumber(item.hit_count || 0)} hint="合并后的命中数" tone="neutral" />
        <StatTile label="分值变化" value={formatNumber(item.score_delta || 0)} hint="单次事件贡献" tone="warning" />
        <StatTile label="首次出现" value={formatDateTime(item.first_seen_at)} hint="first seen" tone="info" />
        <StatTile label="最后出现" value={formatDateTime(item.last_seen_at)} hint="last seen" tone="success" />
      </div>

      <div className="detail-chip-list">
        {renderEventDetail(item.detail).map(([key, value]) => (
          <span key={`${item.id}-${key}`} className="detail-chip">
            <strong>{key}</strong>: {typeof value === "object" ? JSON.stringify(value) : String(value)}
          </span>
        ))}
      </div>
    </div>
  );
}

function EventsPage({
  items,
  total,
  page,
  pageSize,
  filters,
  loading,
  error,
  onRefresh,
  onPageChange,
  onFilterChange,
}) {
  const summary = useMemo(
    () =>
      items.reduce((accumulator, item) => {
        accumulator[item.status] = (accumulator[item.status] || 0) + 1;
        return accumulator;
      }, {}),
    [items]
  );

  return (
    <section className="page-section">
      <SectionHeader
        eyebrow="Events"
        title="风险事件"
        description="事件页按生命周期看，不再只盯 severity。open 是待处理，resolved 是已出窗口的历史事件。"
        actions={<ToolbarButton onClick={onRefresh}>刷新事件</ToolbarButton>}
      />

      <div className="filter-grid filter-grid-wide">
        <FilterField label="用户 ID">
          <input value={filters.user_id} onChange={(event) => onFilterChange("user_id", event.target.value)} placeholder="搜索用户 ID" />
        </FilterField>
        <FilterField label="严重级别">
          <select value={filters.severity} onChange={(event) => onFilterChange("severity", event.target.value)}>
            <option value="">全部级别</option>
            <option value="high">high</option>
            <option value="warning">warning</option>
            <option value="info">info</option>
          </select>
        </FilterField>
        <FilterField label="事件状态">
          <select value={filters.status} onChange={(event) => onFilterChange("status", event.target.value)}>
            <option value="">全部状态</option>
            <option value="open">open</option>
            <option value="resolved">resolved</option>
            <option value="acknowledged">acknowledged</option>
            <option value="ignored">ignored</option>
          </select>
        </FilterField>
        <FilterField label="事件类型">
          <input value={filters.event_type} onChange={(event) => onFilterChange("event_type", event.target.value)} placeholder="如 cost_spike" />
        </FilterField>
      </div>

      <div className="kpi-grid compact-grid">
        <StatTile label="当前结果" value={formatNumber(total)} hint="符合筛选条件的事件" tone="neutral" />
        <StatTile label="Open" value={formatNumber(summary.open || 0)} hint="当前页" tone={(summary.open || 0) > 0 ? "danger" : "success"} />
        <StatTile label="Resolved" value={formatNumber(summary.resolved || 0)} hint="当前页" tone="success" />
        <StatTile label="Acknowledged" value={formatNumber(summary.acknowledged || 0)} hint="当前页" tone="warning" />
      </div>

      <SectionState loading={loading} error={error} empty={!loading && !error && items.length === 0} emptyText="没有命中的风险事件">
        <>
          <div className="event-list">
            {items.map((item) => (
              <EventCard key={`${item.id}-${item.event_fingerprint}`} item={item} />
            ))}
          </div>
          <PaginationBar page={page} pageSize={pageSize} total={total} onPageChange={onPageChange} />
        </>
      </SectionState>
    </section>
  );
}

function PaginationBar({ page, pageSize, total, onPageChange }) {
  const totalPages = Math.max(1, Math.ceil((total || 0) / pageSize));
  const safePage = Math.min(page, totalPages);

  return (
    <div className="pagination-bar">
      <div className="pagination-meta">
        共 {formatNumber(total)} 条 · 第 {safePage} / {totalPages} 页
      </div>
      <div className="pagination-actions">
        <ToolbarButton onClick={() => onPageChange(Math.max(1, safePage - 1))} disabled={safePage <= 1}>
          上一页
        </ToolbarButton>
        <ToolbarButton onClick={() => onPageChange(Math.min(totalPages, safePage + 1))} disabled={safePage >= totalPages}>
          下一页
        </ToolbarButton>
      </div>
    </div>
  );
}

function GameMonitorPage(props) {
  const {
    overview,
    overviewLoading,
    overviewError,
    userItems,
    roundItems,
    claimItems,
    eventItems,
    userTotal,
    roundTotal,
    claimTotal,
    eventTotal,
    filters,
    pageState,
    loadingState,
    errorState,
    onRefresh,
    onFilterChange,
    onPageChange,
  } = props;

  return (
    <section className="page-section">
      <SectionHeader
        eyebrow="Game monitor"
        title="游戏监控"
        description="把开局、揭晓、claim 状态、补偿重试和硬上限命中放到一页里，直接看今天到底发了多少、卡了多少。"
        actions={<ToolbarButton onClick={onRefresh}>刷新游戏监控</ToolbarButton>}
      />

      <div className="filter-grid filter-grid-wide">
        <FilterField label="日期">
          <input value={filters.day_key} onChange={(event) => onFilterChange("day_key", event.target.value)} placeholder="2026-05-13" />
        </FilterField>
        <FilterField label="用户 ID">
          <input value={filters.user_id} onChange={(event) => onFilterChange("user_id", event.target.value)} placeholder="搜索用户 ID" />
        </FilterField>
        <FilterField label="玩法">
          <select value={filters.game_id} onChange={(event) => onFilterChange("game_id", event.target.value)}>
            <option value="">全部玩法</option>
            <option value="flip_card">翻牌</option>
            <option value="lucky_wheel">转盘</option>
            <option value="smash_egg">砸金蛋</option>
          </select>
        </FilterField>
        <FilterField label="模式">
          <select value={filters.risk_mode} onChange={(event) => onFilterChange("risk_mode", event.target.value)}>
            <option value="">全部模式</option>
            <option value="steady">稳</option>
            <option value="high_multiplier">高倍率</option>
          </select>
        </FilterField>
        <FilterField label="Claim 状态">
          <select value={filters.claim_status} onChange={(event) => onFilterChange("claim_status", event.target.value)}>
            <option value="">全部状态</option>
            <option value="succeeded">succeeded</option>
            <option value="reconcile_pending">reconcile_pending</option>
            <option value="failed">failed</option>
            <option value="none">none</option>
          </select>
        </FilterField>
        <FilterField label="事件类型">
          <input value={filters.event_type} onChange={(event) => onFilterChange("event_type", event.target.value)} placeholder="如 cap_hit / redeem_failed" />
        </FilterField>
      </div>

      <SectionState loading={overviewLoading} error={overviewError} empty={!overview} emptyText="暂无游戏监控概览数据">
        <div className="kpi-grid compact-grid">
          <StatTile label="今日总局数" value={formatNumber(overview?.total_rounds || 0)} hint={`日期 ${overview?.day_key || "-"}`} tone="neutral" />
          <StatTile label="免费局数" value={formatNumber(overview?.free_rounds || 0)} hint="今日汇总" tone="success" />
          <StatTile label="付费局数" value={formatNumber(overview?.paid_rounds || 0)} hint="今日汇总" tone="warning" />
          <StatTile label="今日毛奖励" value={formatMoney(overview?.gross_reward_amount || 0)} hint="揭晓结果总和" tone="info" />
          <StatTile label="今日入场费" value={formatMoney(overview?.entry_fee_amount || 0)} hint="购次扣费" tone="warning" />
          <StatTile label="今日净发放" value={formatSignedMoney(overview?.net_reward_amount || 0)} hint="真实净变化" tone={Number(overview?.net_reward_amount || 0) >= 0 ? "success" : "danger"} />
          <StatTile label="Pending Claims" value={formatNumber(overview?.pending_claims || 0)} hint="待补偿/待同步" tone={(overview?.pending_claims || 0) > 0 ? "warning" : "success"} />
          <StatTile label="Cap Hits" value={formatNumber(overview?.cap_hits || 0)} hint={`刷新 ${formatRelativeTime(overview?.refreshed_at)}`} tone={(overview?.cap_hits || 0) > 0 ? "warning" : "neutral"} />
        </div>
      </SectionState>

      <div className="content-grid content-grid-2">
        <div className="panel-card">
          <div className="panel-card-head">
            <div className="panel-card-title">用户榜单</div>
            <div className="panel-card-subtitle">净收益 Top、游玩次数 Top、待补偿用户。</div>
          </div>
          <SectionState loading={loadingState.users} error={errorState.users} empty={!userItems.length} emptyText="暂无用户榜单">
            <div className="table-shell">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>用户</th>
                    <th>已玩</th>
                    <th>免费/付费</th>
                    <th>净收益</th>
                    <th>待补偿</th>
                    <th>失败</th>
                  </tr>
                </thead>
                <tbody>
                  {userItems.map((item) => (
                    <tr key={`${item.user_id}-${item.day_key}`}>
                      <td>
                        <div className="table-primary">{item.username || item.user_id}</div>
                        <div className="table-secondary">{item.user_id}</div>
                      </td>
                      <td>{formatNumber(item.play_count)}</td>
                      <td>{formatNumber(item.free_play_count)} / {formatNumber(item.paid_play_count)}</td>
                      <td>{formatSignedMoney(item.net_reward_amount)}</td>
                      <td>{formatNumber(item.pending_claim_count)}</td>
                      <td>{formatNumber(item.failed_claim_count)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <PaginationBar page={pageState.users} pageSize={20} total={userTotal} onPageChange={(value) => onPageChange("users", value)} />
          </SectionState>
        </div>

        <div className="panel-card">
          <div className="panel-card-head">
            <div className="panel-card-title">异常事件</div>
            <div className="panel-card-subtitle">redeem 失败、补偿失败、cap 命中和异常频发用户都会落这里。</div>
          </div>
          <SectionState loading={loadingState.events} error={errorState.events} empty={!eventItems.length} emptyText="暂无异常事件">
            <div className="event-list">
              {eventItems.slice(0, 8).map((item) => (
                <EventCard
                  key={item.event_key}
                  item={{
                    id: item.event_key,
                    title: item.message || item.event_type,
                    event_type: item.event_type,
                    user_id: item.user_id,
                    event_fingerprint: item.event_key,
                    status: "open",
                    severity: item.severity,
                    hit_count: 1,
                    score_delta: 0,
                    first_seen_at: item.occurred_at,
                    last_seen_at: item.occurred_at,
                    detail: item.detail,
                  }}
                />
              ))}
            </div>
            <PaginationBar page={pageState.events} pageSize={20} total={eventTotal} onPageChange={(value) => onPageChange("events", value)} />
          </SectionState>
        </div>
      </div>

      <div className="panel-card">
        <div className="panel-card-head">
          <div className="panel-card-title">Round 明细</div>
          <div className="panel-card-subtitle">逐局看开局、揭晓、模式、付费标记和净变化。</div>
        </div>
        <SectionState loading={loadingState.rounds} error={errorState.rounds} empty={!roundItems.length} emptyText="暂无 round 明细">
          <div className="table-shell">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Round</th>
                  <th>用户</th>
                  <th>玩法</th>
                  <th>模式</th>
                  <th>状态</th>
                  <th>净变化</th>
                  <th>创建时间</th>
                </tr>
              </thead>
              <tbody>
                {roundItems.map((item) => (
                  <tr key={item.round_id}>
                    <td className="table-wrap">{item.round_id}</td>
                    <td>{item.username || item.user_id}</td>
                    <td>{item.game_id}</td>
                    <td>{item.risk_mode}</td>
                    <td><Pill tone={eventStatusTone(item.claim_status === "none" ? item.status : item.claim_status)}>{item.claim_status === "none" ? item.status : item.claim_status}</Pill></td>
                    <td>{formatSignedMoney(item.net_reward_amount)}</td>
                    <td>{formatDateTime(item.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <PaginationBar page={pageState.rounds} pageSize={20} total={roundTotal} onPageChange={(value) => onPageChange("rounds", value)} />
        </SectionState>
      </div>

      <div className="panel-card">
        <div className="panel-card-head">
          <div className="panel-card-title">Claim 明细</div>
          <div className="panel-card-subtitle">逐条看扣费 claim、结算 claim、重试次数和最终状态。</div>
        </div>
        <SectionState loading={loadingState.claims} error={errorState.claims} empty={!claimItems.length} emptyText="暂无 claim 明细">
          <div className="table-shell">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Claim</th>
                  <th>用户</th>
                  <th>玩法</th>
                  <th>类型</th>
                  <th>状态</th>
                  <th>净变化</th>
                  <th>重试/错误</th>
                </tr>
              </thead>
              <tbody>
                {claimItems.map((item) => (
                  <tr key={item.claim_id}>
                    <td className="table-wrap">{item.claim_id}</td>
                    <td>{item.username || item.user_id}</td>
                    <td>{item.game_id}</td>
                    <td>{item.claim_kind}</td>
                    <td><Pill tone={eventStatusTone(item.claim_status)}>{item.claim_status}</Pill></td>
                    <td>{formatSignedMoney(item.net_reward_amount)}</td>
                    <td>
                      <div className="table-primary">{formatNumber(item.attempt_count)}</div>
                      <div className="table-secondary">{item.last_error || "-"}</div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <PaginationBar page={pageState.claims} pageSize={20} total={claimTotal} onPageChange={(value) => onPageChange("claims", value)} />
        </SectionState>
      </div>
    </section>
  );
}

function App() {
  const [activeTab, setActiveTab] = useState(defaultTab);

  const [meta, setMeta] = useState(null);
  const [metaError, setMetaError] = useState("");

  const [overview, setOverview] = useState(null);
  const [overviewLoading, setOverviewLoading] = useState(true);
  const [overviewError, setOverviewError] = useState("");

  const [operationsOverview, setOperationsOverview] = useState(null);
  const [operationsLoading, setOperationsLoading] = useState(true);
  const [operationsError, setOperationsError] = useState("");

  const [userFilters, setUserFilters] = useState({
    user_id: "",
    risk_level: "",
    include_stale: false,
  });
  const [userPage, setUserPage] = useState(1);
  const [userPageSize] = useState(20);
  const [users, setUsers] = useState([]);
  const [usersTotal, setUsersTotal] = useState(0);
  const [usersLoading, setUsersLoading] = useState(true);
  const [usersError, setUsersError] = useState("");

  const [eventFilters, setEventFilters] = useState({
    user_id: "",
    severity: "",
    status: "open",
    event_type: "",
  });
  const [eventPage, setEventPage] = useState(1);
  const [eventPageSize] = useState(20);
  const [events, setEvents] = useState([]);
  const [eventsTotal, setEventsTotal] = useState(0);
  const [eventsLoading, setEventsLoading] = useState(true);
  const [eventsError, setEventsError] = useState("");
  const [gameOverview, setGameOverview] = useState(null);
  const [gameOverviewLoading, setGameOverviewLoading] = useState(true);
  const [gameOverviewError, setGameOverviewError] = useState("");
  const [gameFilters, setGameFilters] = useState({
    day_key: "",
    user_id: "",
    game_id: "",
    risk_mode: "",
    claim_status: "",
    event_type: "",
  });
  const [gamePageState, setGamePageState] = useState({
    users: 1,
    rounds: 1,
    claims: 1,
    events: 1,
  });
  const [gameUsers, setGameUsers] = useState([]);
  const [gameUsersTotal, setGameUsersTotal] = useState(0);
  const [gameUsersLoading, setGameUsersLoading] = useState(true);
  const [gameUsersError, setGameUsersError] = useState("");
  const [gameRounds, setGameRounds] = useState([]);
  const [gameRoundsTotal, setGameRoundsTotal] = useState(0);
  const [gameRoundsLoading, setGameRoundsLoading] = useState(true);
  const [gameRoundsError, setGameRoundsError] = useState("");
  const [gameClaims, setGameClaims] = useState([]);
  const [gameClaimsTotal, setGameClaimsTotal] = useState(0);
  const [gameClaimsLoading, setGameClaimsLoading] = useState(true);
  const [gameClaimsError, setGameClaimsError] = useState("");
  const [gameEvents, setGameEvents] = useState([]);
  const [gameEventsTotal, setGameEventsTotal] = useState(0);
  const [gameEventsLoading, setGameEventsLoading] = useState(true);
  const [gameEventsError, setGameEventsError] = useState("");

  async function loadMeta() {
    try {
      setMetaError("");
      const data = await fetchJson("/meta");
      setMeta(data);
    } catch (error) {
      setMetaError(error.message);
    }
  }

  async function loadOverview() {
    try {
      setOverviewLoading(true);
      setOverviewError("");
      const data = await fetchJson("/dashboard/overview");
      setOverview(data);
    } catch (error) {
      setOverviewError(error.message);
    } finally {
      setOverviewLoading(false);
    }
  }

  async function loadOperationsOverview() {
    try {
      setOperationsLoading(true);
      setOperationsError("");
      const data = await fetchJson("/operations/overview");
      setOperationsOverview(data);
    } catch (error) {
      setOperationsError(error.message);
    } finally {
      setOperationsLoading(false);
    }
  }

  async function loadUsers() {
    try {
      setUsersLoading(true);
      setUsersError("");
      const data = await fetchJson("/users", {
        page: userPage,
        page_size: userPageSize,
        user_id: userFilters.user_id,
        risk_level: userFilters.risk_level,
        include_stale: userFilters.include_stale ? 1 : 0,
      });
      setUsers(data.items || []);
      setUsersTotal(data.total || 0);
    } catch (error) {
      setUsersError(error.message);
    } finally {
      setUsersLoading(false);
    }
  }

  async function loadEvents() {
    try {
      setEventsLoading(true);
      setEventsError("");
      const data = await fetchJson("/events", {
        page: eventPage,
        page_size: eventPageSize,
        user_id: eventFilters.user_id,
        severity: eventFilters.severity,
        status: eventFilters.status,
        event_type: eventFilters.event_type,
      });
      setEvents(data.items || []);
      setEventsTotal(data.total || 0);
    } catch (error) {
      setEventsError(error.message);
    } finally {
      setEventsLoading(false);
    }
  }

  async function loadGameOverview() {
    try {
      setGameOverviewLoading(true);
      setGameOverviewError("");
      const data = await fetchJson("/game/overview", {
        day_key: gameFilters.day_key,
      });
      setGameOverview(data);
    } catch (error) {
      setGameOverviewError(error.message);
    } finally {
      setGameOverviewLoading(false);
    }
  }

  async function loadGameUsers() {
    try {
      setGameUsersLoading(true);
      setGameUsersError("");
      const data = await fetchJson("/game/users", {
        page: gamePageState.users,
        page_size: 20,
        day_key: gameFilters.day_key,
        user_id: gameFilters.user_id,
        sort_by: "net_reward_amount",
      });
      setGameUsers(data.items || []);
      setGameUsersTotal(data.total || 0);
    } catch (error) {
      setGameUsersError(error.message);
    } finally {
      setGameUsersLoading(false);
    }
  }

  async function loadGameRounds() {
    try {
      setGameRoundsLoading(true);
      setGameRoundsError("");
      const data = await fetchJson("/game/rounds", {
        page: gamePageState.rounds,
        page_size: 20,
        day_key: gameFilters.day_key,
        user_id: gameFilters.user_id,
        game_id: gameFilters.game_id,
        risk_mode: gameFilters.risk_mode,
        claim_status: gameFilters.claim_status,
      });
      setGameRounds(data.items || []);
      setGameRoundsTotal(data.total || 0);
    } catch (error) {
      setGameRoundsError(error.message);
    } finally {
      setGameRoundsLoading(false);
    }
  }

  async function loadGameClaims() {
    try {
      setGameClaimsLoading(true);
      setGameClaimsError("");
      const data = await fetchJson("/game/claims", {
        page: gamePageState.claims,
        page_size: 20,
        day_key: gameFilters.day_key,
        user_id: gameFilters.user_id,
        game_id: gameFilters.game_id,
        risk_mode: gameFilters.risk_mode,
        claim_status: gameFilters.claim_status,
      });
      setGameClaims(data.items || []);
      setGameClaimsTotal(data.total || 0);
    } catch (error) {
      setGameClaimsError(error.message);
    } finally {
      setGameClaimsLoading(false);
    }
  }

  async function loadGameEvents() {
    try {
      setGameEventsLoading(true);
      setGameEventsError("");
      const data = await fetchJson("/game/events", {
        page: gamePageState.events,
        page_size: 20,
        day_key: gameFilters.day_key,
        user_id: gameFilters.user_id,
        game_id: gameFilters.game_id,
        risk_mode: gameFilters.risk_mode,
        event_type: gameFilters.event_type,
      });
      setGameEvents(data.items || []);
      setGameEventsTotal(data.total || 0);
    } catch (error) {
      setGameEventsError(error.message);
    } finally {
      setGameEventsLoading(false);
    }
  }

  useEffect(() => {
    loadMeta();
    loadOverview();
    loadOperationsOverview();
    loadGameOverview();
  }, []);

  useEffect(() => {
    loadUsers();
  }, [userPage, userPageSize, userFilters.user_id, userFilters.risk_level, userFilters.include_stale]);

  useEffect(() => {
    loadEvents();
  }, [eventPage, eventPageSize, eventFilters.user_id, eventFilters.severity, eventFilters.status, eventFilters.event_type]);

  useEffect(() => {
    loadGameOverview();
    loadGameUsers();
    loadGameRounds();
    loadGameClaims();
    loadGameEvents();
  }, [
    gameFilters.day_key,
    gameFilters.user_id,
    gameFilters.game_id,
    gameFilters.risk_mode,
    gameFilters.claim_status,
    gameFilters.event_type,
    gamePageState.users,
    gamePageState.rounds,
    gamePageState.claims,
    gamePageState.events,
  ]);

  useEffect(() => {
    loadGameOverview();
    loadGameUsers();
    loadGameRounds();
    loadGameClaims();
    loadGameEvents();
  }, [
    gameFilters.day_key,
    gameFilters.user_id,
    gameFilters.game_id,
    gameFilters.risk_mode,
    gameFilters.claim_status,
    gameFilters.event_type,
    gamePageState.users,
    gamePageState.rounds,
    gamePageState.claims,
    gamePageState.events,
  ]);

  const tabs = [
    { key: "operations", label: "运营统计" },
    { key: "overview", label: "风控总览" },
    { key: "users", label: "风险用户" },
    { key: "events", label: "风险事件" },
  ];

  return (
    <div className={`app-shell ${forceEmbed ? "app-shell-embed" : ""}`}>
      <div className="topbar">
        <div>
          <div className="topbar-eyebrow">sub2api risk center</div>
          <h1 className="topbar-title">风控中心</h1>
          <p className="topbar-description">
            把实时风险、运营转化和资源消耗放进一个工作台里，先保证结论清楚，再谈策略。
          </p>
        </div>
        <div className="topbar-actions">
          <ToolbarButton primary onClick={() => {
            loadMeta();
            loadOverview();
            loadOperationsOverview();
            loadUsers();
            loadEvents();
            loadGameOverview();
            loadGameUsers();
            loadGameRounds();
            loadGameClaims();
            loadGameEvents();
          }}>
            全量刷新
          </ToolbarButton>
        </div>
      </div>

      <div className="meta-strip">
        <Pill tone={meta?.read_only_mode ? "warning" : "success"}>{meta?.read_only_mode ? "只读模式" : "可刷新"}</Pill>
        <Pill tone={freshnessTone(overview?.last_refresh_at)}>{formatRelativeTime(overview?.last_refresh_at)}</Pill>
        <Pill tone="neutral">
          窗口 {formatDateTime(overview?.window_start)} - {formatDateTime(overview?.window_end)}
        </Pill>
        {metaError ? <Pill tone="danger">{metaError}</Pill> : null}
      </div>

      <div className="tab-strip">
        {tabs.map((tab) => (
          <TabButton key={tab.key} active={activeTab === tab.key} onClick={() => setActiveTab(tab.key)}>
            {tab.label}
          </TabButton>
        ))}
        <TabButton active={activeTab === "game"} onClick={() => setActiveTab("game")}>
          游戏监控
        </TabButton>
      </div>

      <div className="workspace">
        {activeTab === "operations" ? (
          <OperationsPage
            overview={operationsOverview}
            loading={operationsLoading}
            error={operationsError}
            onRefresh={loadOperationsOverview}
          />
        ) : null}

        {activeTab === "overview" ? (
          <OverviewPage overview={overview} loading={overviewLoading} error={overviewError} onRefresh={loadOverview} />
        ) : null}

        {activeTab === "users" ? (
          <UsersPage
            items={users}
            total={usersTotal}
            page={userPage}
            pageSize={userPageSize}
            filters={userFilters}
            loading={usersLoading}
            error={usersError}
            onRefresh={loadUsers}
            onPageChange={setUserPage}
            onFilterChange={(key, value) => {
              setUserPage(1);
              setUserFilters((current) => ({ ...current, [key]: value }));
            }}
          />
        ) : null}

        {activeTab === "events" ? (
          <EventsPage
            items={events}
            total={eventsTotal}
            page={eventPage}
            pageSize={eventPageSize}
            filters={eventFilters}
            loading={eventsLoading}
            error={eventsError}
            onRefresh={loadEvents}
            onPageChange={setEventPage}
            onFilterChange={(key, value) => {
              setEventPage(1);
              setEventFilters((current) => ({ ...current, [key]: value }));
            }}
          />
        ) : null}
        {activeTab === "game" ? (
          <GameMonitorPage
            overview={gameOverview}
            overviewLoading={gameOverviewLoading}
            overviewError={gameOverviewError}
            userItems={gameUsers}
            roundItems={gameRounds}
            claimItems={gameClaims}
            eventItems={gameEvents}
            userTotal={gameUsersTotal}
            roundTotal={gameRoundsTotal}
            claimTotal={gameClaimsTotal}
            eventTotal={gameEventsTotal}
            filters={gameFilters}
            pageState={gamePageState}
            loadingState={{
              users: gameUsersLoading,
              rounds: gameRoundsLoading,
              claims: gameClaimsLoading,
              events: gameEventsLoading,
            }}
            errorState={{
              users: gameUsersError,
              rounds: gameRoundsError,
              claims: gameClaimsError,
              events: gameEventsError,
            }}
            onRefresh={() => {
              loadGameOverview();
              loadGameUsers();
              loadGameRounds();
              loadGameClaims();
              loadGameEvents();
            }}
            onFilterChange={(key, value) => {
              setGamePageState({ users: 1, rounds: 1, claims: 1, events: 1 });
              setGameFilters((current) => ({ ...current, [key]: value }));
            }}
            onPageChange={(key, value) => {
              setGamePageState((current) => ({ ...current, [key]: value }));
            }}
          />
        ) : null}
      </div>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
