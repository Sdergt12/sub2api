import React, { useEffect, useMemo, useState } from "react";
import ReactDOM from "react-dom/client";

const SEARCH_PARAMS = new URLSearchParams(window.location.search);
const EMBED_TOKEN = SEARCH_PARAMS.get("k") || "";
const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL || "/api/sign").replace(/\/$/, "");
const BOOTSTRAP_URL = import.meta.env.VITE_BOOTSTRAP_URL || "/embed/bootstrap";
const ASSET_BASE_URL = `${import.meta.env.BASE_URL || "/"}sign-assets/`;
const ASSET_VERSION = "20260418-ui-2";

function buildHeaders(extraHeaders) {
  const headers = {
    Accept: "application/json",
    ...extraHeaders,
  };

  if (EMBED_TOKEN) {
    headers["X-Sign-Embed-Token"] = EMBED_TOKEN;
  }

  return headers;
}

async function fetchJson(path, options = {}) {
  const response = await fetch(path.startsWith("http") ? path : `${API_BASE_URL}${path}`, {
    method: options.method || "GET",
    headers: buildHeaders(options.headers),
    credentials: "include",
    body: options.body,
  });

  if (!response.ok) {
    let message = `HTTP ${response.status}`;
    try {
      const payload = await response.json();
      if (payload?.error) {
        message = payload.error;
      }
    } catch {
      // keep default message
    }
    throw new Error(message);
  }

  return response.json();
}

async function bootstrapSession() {
  const query = new URLSearchParams(window.location.search);
  const separator = BOOTSTRAP_URL.includes("?") ? "&" : "?";
  const response = await fetch(`${BOOTSTRAP_URL}${separator}${query.toString()}`, {
    method: "GET",
    headers: buildHeaders(),
    credentials: "include",
  });

  if (!response.ok) {
    let message = `HTTP ${response.status}`;
    try {
      const payload = await response.json();
      if (payload?.error) {
        message = payload.error;
      }
    } catch {
      // keep default message
    }
    throw new Error(message);
  }

  return response.json();
}

function App() {
  const [bootData, setBootData] = useState(null);
  const [statusData, setStatusData] = useState(null);
  const [checkingIn, setCheckingIn] = useState(false);
  const [rewardPopup, setRewardPopup] = useState(null);
  const [error, setError] = useState("");
  const [pageLoading, setPageLoading] = useState(true);
  const [rulesOpen, setRulesOpen] = useState(false);

  const effectiveTheme = useMemo(() => bootData?.theme || SEARCH_PARAMS.get("theme") || "light", [bootData]);
  const darkMode = effectiveTheme === "dark";
  const palette = getPalette(darkMode);

  async function reloadStatus() {
    const status = await fetchJson("/status");
    setStatusData(status);
    return status;
  }

  useEffect(() => {
    let cancelled = false;

    async function init() {
      try {
        setPageLoading(true);
        setError("");
        const boot = await bootstrapSession();
        if (cancelled) {
          return;
        }
        setBootData(boot);
        setStatusData({
          signed_today: boot.signed_today,
          streak: boot.streak,
          today_reward_min: boot.today_reward_min,
          today_reward_max: boot.today_reward_max,
          current_balance_display: boot.current_balance_display,
          recent_sign_days: boot.recent_sign_days,
          month_sign_days: boot.month_sign_days,
          history: boot.history,
        });
      } catch (initError) {
        if (!cancelled) {
          setError(initError.message || "初始化失败");
        }
      } finally {
        if (!cancelled) {
          setPageLoading(false);
        }
      }
    }

    init();
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleCheckin() {
    try {
      setCheckingIn(true);
      setError("");
      const result = await fetchJson("/checkin", { method: "POST" });
      const refreshed = await reloadStatus();
      setRewardPopup({
        totalReward: result.total_reward,
        streak: result.streak,
        balance: result.current_balance_display,
      });

      setStatusData({
        ...refreshed,
        current_balance_display: result.current_balance_display || refreshed.current_balance_display,
      });

      window.setTimeout(() => setRewardPopup(null), 2200);
    } catch (checkinError) {
      setError(checkinError.message || "签到失败");
    } finally {
      setCheckingIn(false);
    }
  }

  const signedToday = Boolean(statusData?.signed_today);
  const streak = statusData?.streak || 0;
  const history = statusData?.history || [];
  const todayKey = getTodayKey();
  const historyByDate = new Map(history.map((item) => [item.sign_date, item]));
  const todayReward = history.find((item) => item.sign_date === todayKey && item.status === "success");
  const displayReward = todayReward || (signedToday ? history.find((item) => item.status === "success") : null);
  const todaySignedAt = todayReward?.created_at || displayReward?.created_at;
  const heroTitle = pageLoading ? "正在读取签到状态" : signedToday ? "签到成功！" : "今日签到可领取奖励";
  const heroDescription = signedToday
    ? "今日奖励已到账，明天继续保持连续签到。"
    : "每日可随机获得额度，连续签到有额外加成。";
  const rewardRangeText = `${statusData?.today_reward_min || "--"} - ${statusData?.today_reward_max || "--"}`;
  const recentSignDays = new Set(statusData?.recent_sign_days || []);
  const monthSignDays = new Set(statusData?.month_sign_days || []);
  const currentMonthDays = buildCurrentMonthDays();
  const signedMonthCount = monthSignDays.size;
  const nextMonthlyTarget = [3, 7, 15, 30].find((target) => signedMonthCount < target) || 30;
  const daysToMonthlyTarget = Math.max(nextMonthlyTarget - signedMonthCount, 0);
  const lastSevenDays = Array.from({ length: 7 }, (_, index) => {
    const date = new Date();
    date.setHours(0, 0, 0, 0);
    date.setDate(date.getDate() - (6 - index));
    const yyyy = date.getFullYear();
    const mm = String(date.getMonth() + 1).padStart(2, "0");
    const dd = String(date.getDate()).padStart(2, "0");
    return `${yyyy}-${mm}-${dd}`;
  });

  return (
    <div style={{ ...pageStyle, background: palette.pageBackground, color: palette.textPrimary }}>
      <style>{animationStyles}</style>
      <div style={{ ...ambientBandStyle, background: palette.ambientBackground }} />
      <div style={pageInnerStyle}>
        <section style={{ ...mainCardStyle, background: palette.mainCardBackground, borderColor: palette.softBorder, boxShadow: palette.cardShadow }}>
          <div style={cardGlowStyle} />
          <div style={mainCardContentStyle}>
            <div style={mainTopStyle}>
              <div style={heroTextColumnStyle}>
                <div style={heroCopyStyle}>
                  <div style={{ ...eyebrowStyle, color: palette.textSecondary }}>每日签到</div>
                  <h1 style={heroTitleStyle}>{heroTitle}</h1>
                  <p style={{ ...heroDescStyle, color: palette.textSecondary }}>
                    {heroDescription}
                  </p>
                </div>

                <div style={rewardFocusStyle}>
                  {signedToday && displayReward ? (
                    <div style={rewardCenterStyle}>
                      <div style={{ ...coinIconStyle, background: palette.rewardSoft, color: palette.rewardText, borderColor: palette.rewardBorder }}>
                        <IconMask name="icon-coin.svg" size={24} />
                      </div>
                      <div style={{ ...rewardCenterAmountStyle, color: palette.rewardText, textShadow: palette.rewardGlow }}>
                        +{formatAmount(displayReward.total_reward)}
                      </div>
                      <div style={{ ...rewardCenterStatusStyle, color: palette.successText, background: palette.successSoft }}>已到账</div>
                      {todaySignedAt ? (
                        <div style={{ marginTop: 8, fontSize: 12, color: palette.textMuted }}>
                          领取时间 {formatDateTime(todaySignedAt)}
                        </div>
                      ) : null}
                    </div>
                  ) : (
                    <div style={pendingRewardStyle}>
                      <div style={{ ...giftIconStyle, background: palette.rewardSoft, color: palette.rewardText }}>
                        <IconMask name="icon-gift.svg" size={20} />
                      </div>
                      <div style={{ color: palette.textSecondary }}>点击签到后由服务端判定今日奖励</div>
                    </div>
                  )}
                </div>
              </div>

              <div className="hero-illustration" style={heroIllustrationStyle} aria-hidden="true">
                <div style={heroIllustrationShellStyle}>
                  <img
                    src={assetUrl(signedToday ? "reward-winners.svg" : "reward-wallet.svg")}
                    alt=""
                    style={heroImageStyle}
                    loading="eager"
                  />
                </div>
              </div>
            </div>

            <div className="mobile-stack" style={mainStatsStyle}>
              <MetricBlock label="连续签到" value={`${streak}`} suffix="天" note="已连续签到" palette={palette} />
              <MetricBlock
                label="当前余额"
                value={formatAmount(statusData?.current_balance_display || "--", { compact: true })}
                fullValue={formatAmount(statusData?.current_balance_display || "--")}
                suffix=""
                note="剩余额度"
                palette={palette}
                strong
              />
            </div>

            <div style={{ ...rewardHintStyle, color: palette.textSecondary, background: palette.infoSoft }}>
              <span style={{ color: palette.infoText }}>i</span>
              基础随机奖励范围：<strong style={{ color: palette.textPrimary }}>{rewardRangeText}</strong>
              <span style={{ color: palette.textMuted }}> · 明日刷新后重新判定</span>
            </div>

            <div style={primaryActionRowStyle}>
              <button
                type="button"
                onClick={handleCheckin}
                disabled={pageLoading || checkingIn || signedToday}
                style={{
                  ...primaryButtonStyle,
                  background: signedToday ? palette.successButton : palette.buttonPrimary,
                  color: signedToday ? palette.successButtonText : "#ffffff",
                  boxShadow: signedToday ? "none" : palette.buttonGlow,
                  animation: !signedToday && !checkingIn ? "buttonAura 2.6s ease-in-out infinite" : "none",
                  cursor: signedToday ? "default" : "pointer",
                }}
              >
                {checkingIn ? "奖励发放中..." : signedToday ? "✓ 今日奖励已到账" : "立即签到"}
              </button>
              <button type="button" onClick={reloadStatus} aria-label="刷新数据" style={{ ...secondaryButtonStyle, borderColor: palette.softBorder, color: palette.textSecondary }}>
                ↻ 刷新数据
              </button>
            </div>

            {error ? (
              <div style={{ ...alertStyle, background: palette.alertBackground, color: palette.alertText, borderColor: palette.alertBorder }}>{error}</div>
            ) : null}
          </div>
        </section>

        <section style={sectionStyle}>
          <div style={sectionHeaderStyle}>
            <div>
              <h2 style={sectionTitleStyle}>本月签到概览</h2>
              <p style={{ ...sectionDescStyle, color: palette.textSecondary }}>累计签到越稳定，越接近月度奖励节点。</p>
            </div>
          </div>
          <div className="mobile-stack" style={monthOverviewStyle}>
            <div style={{ ...monthSummaryStyle, background: palette.panelGlass, boxShadow: palette.listShadow }}>
              <div style={monthSummaryHeroStyle}>
                <div>
                  <div style={{ fontSize: 12, fontWeight: 700, color: palette.textMuted }}>月度目标</div>
                  <div style={{ marginTop: 6, fontSize: 15, fontWeight: 700, color: palette.textPrimary }}>稳定签到更容易触发奖励节点</div>
                </div>
                <img src={assetUrl("reward-calendar.svg")} alt="" aria-hidden="true" style={monthSummaryImageStyle} />
              </div>
              <MetricBlock label="本月已签到" value={`${signedMonthCount}`} suffix="天" note={`距离 ${nextMonthlyTarget} 天奖励还差 ${daysToMonthlyTarget} 天`} palette={palette} />
              <MetricBlock label="当前连签" value={`${streak}`} suffix="天" note="连续目标进行中" palette={palette} />
            </div>
            <div style={{ ...monthCalendarStyle, background: palette.panelGlass, boxShadow: palette.listShadow }}>
              {currentMonthDays.map((day) => {
                const signed = monthSignDays.has(day.dateKey);
                const isToday = day.dateKey === todayKey;
                const milestone = [3, 7, 15, 30].includes(day.day);
                return (
                  <div
                    key={day.dateKey}
                    title={`${day.dateKey}${milestone ? " 奖励节点" : ""}`}
                    style={{
                      ...monthDayStyle,
                      background: signed ? palette.successSoft : milestone ? palette.rewardSoft : palette.monthDayIdle,
                      color: signed ? palette.successText : milestone ? palette.rewardText : palette.textSecondary,
                      boxShadow: isToday ? palette.todayGlow : "none",
                      borderColor: isToday ? palette.buttonPrimary : "transparent",
                    }}
                  >
                    <span>{day.day}</span>
                    <small style={monthDayMarkStyle}>
                      {signed ? "✓" : milestone ? <IconMask name="icon-gift.svg" size={10} /> : ""}
                    </small>
                  </div>
                );
              })}
            </div>
          </div>
        </section>

        <section style={{ ...progressSectionStyle, borderColor: palette.border }}>
          <div style={sectionHeaderStyle}>
            <div>
              <h2 style={sectionTitleStyle}>近 7 天签到进度</h2>
              <p style={{ ...sectionDescStyle, color: palette.textSecondary }}>点亮连续进度，关键天数会触发额外加成。</p>
            </div>
          </div>
          <div style={timelineGridStyle}>
              {lastSevenDays.map((date, index) => {
                const signed = recentSignDays.has(date);
                const signedAt = historyByDate.get(date)?.created_at;
                const isToday = date === lastSevenDays[lastSevenDays.length - 1];
                const nextDate = lastSevenDays[index + 1];
                const lineActive = signed && Boolean(nextDate && recentSignDays.has(nextDate));
                return (
                  <div
                    key={date}
                    style={{
                      ...timelineItemStyle,
                      color: signed ? palette.successText : palette.textSecondary,
                    }}
                  >
                    <div
                      style={{
                        ...timelineDotStyle,
                        borderColor: signed ? palette.successText : isToday ? palette.buttonPrimary : palette.softBorder,
                        background: signed ? palette.progressNodeActive : palette.progressNodeIdle,
                        boxShadow: isToday ? palette.todayGlow : "none",
                        transform: signed ? "translateY(-2px)" : "none",
                        animation: signed ? "timelinePulse 520ms ease-out both" : isToday ? "todayHalo 2.8s ease-in-out infinite" : "none",
                      }}
                    >
                      <span style={{ fontSize: signed ? 19 : 15, fontWeight: 900 }}>
                        {signed ? "✓" : <IconMask name={index === 2 || index === 6 ? "icon-gift.svg" : "icon-coin.svg"} size={18} />}
                      </span>
                    </div>
                    <div style={timelineDateStyle}>{date.slice(5)}</div>
                    <div style={timelineLabelStyle}>{signed ? "已签" : "未签"}</div>
                    {signedAt ? (
                      <div style={{ fontSize: 11, color: palette.textMuted, marginTop: 2 }}>
                        {formatTimeOnly(signedAt)}
                      </div>
                    ) : null}
                    <span
                      style={{
                        ...timelineLineStyle,
                        background: lineActive ? palette.progressLineActive : palette.border,
                        display: date === lastSevenDays[lastSevenDays.length - 1] ? "none" : "block",
                        opacity: lineActive ? 1 : 0.56,
                        transform: lineActive ? "scaleX(1)" : "scaleX(0.92)",
                      }}
                    />
                  </div>
                );
              })}
            </div>
        </section>

        <section style={sectionStyle}>
          <div style={sectionHeaderStyle}>
            <div>
              <h2 style={sectionTitleStyle}>奖励到账流水</h2>
              <p style={{ ...sectionDescStyle, color: palette.textSecondary }}>展示最近签到奖励、拆分金额与到账状态。</p>
            </div>
          </div>
          <div style={{ display: "grid", gap: 10 }}>
              {history.length === 0 ? (
                <EmptyState text={pageLoading ? "正在加载..." : "还没有签到记录"} palette={palette} />
              ) : (
                history.slice(0, 6).map((item) => (
                  <div key={item.id} className="mobile-stack ledger-row" style={{ ...ledgerItemStyle, background: palette.panelGlass, boxShadow: palette.listShadow }}>
                    <div style={ledgerLeadStyle}>
                      <span style={{ ...ledgerIconStyle, background: palette.rewardSoft, color: palette.rewardText }}>
                        <IconMask name="icon-gift.svg" size={18} />
                      </span>
                      <div>
                        <div style={ledgerTitleStyle}>每日签到奖励</div>
                        <div style={{ fontSize: 13, color: palette.textSecondary }}>{item.sign_date}</div>
                        {item.created_at ? (
                          <div style={{ marginTop: 3, fontSize: 12, color: palette.textMuted }}>到账时间 {formatDateTime(item.created_at)}</div>
                        ) : null}
                      </div>
                    </div>
                    <div style={{ ...ledgerSplitStyle, color: palette.textSecondary }}>
                      <div style={ledgerSummaryTextStyle}>系统结算后自动到账，可在余额流水中追踪。</div>
                      <div style={ledgerSplitRowStyle}>
                        <span style={{ ...splitTagStyle, background: palette.infoSoft }}>基础 +{formatAmount(item.base_reward)}</span>
                        <span style={{ ...splitTagStyle, background: item.bonus_reward !== "0.00" ? palette.rewardSoft : palette.infoSoft, color: item.bonus_reward !== "0.00" ? palette.rewardText : palette.textSecondary }}>
                          连签 +{formatAmount(item.bonus_reward)}
                        </span>
                      </div>
                    </div>
                    <div style={ledgerAmountWrapStyle}>
                      <div style={{ fontSize: 28, fontWeight: 800, color: palette.successText }}>+{formatAmount(item.total_reward)}</div>
                      <div style={{ ...statusPillStyle, background: palette.successSoft, color: palette.successText }}>{formatStatus(item.status)}</div>
                    </div>
                  </div>
                ))
              )}
            </div>
        </section>

        <section style={{ ...rulesSectionStyle, borderColor: palette.border }}>
          <button
            type="button"
            onClick={() => setRulesOpen((open) => !open)}
            style={{ ...rulesToggleStyle, color: palette.textPrimary }}
          >
            <span>签到规则</span>
            <span style={{ ...rulesActionStyle, borderColor: palette.border, color: palette.textSecondary }}>{rulesOpen ? "收起规则" : "查看完整规则"}</span>
          </button>
          <div style={{ ...rulesSummaryStyle, color: palette.textSecondary }}>
            每天一次 · 随机奖励 · 连签有加成
          </div>
          {rulesOpen ? (
            <div style={rulesGridStyle}>
              <RuleItem icon="icon-calendar.svg" title="每天可领一次" text="每天可领取 1 次签到奖励，重复点击不会重复到账。" palette={palette} />
              <RuleItem icon="icon-wallet.svg" title="奖励自动到账" text="签到成功后，奖励会自动发放到账户，并同步记录到奖励流水。" palette={palette} />
              <RuleItem icon="icon-medal.svg" title="连续签到有加成" text="连续签到到第 3、7、15、30 天时，可获得额外奖励，具体金额以当天结果为准。" palette={palette} />
              <RuleItem icon="icon-shield.svg" title="异常请求无效" text="为保障公平，异常频繁或异常来源的请求可能无法获得奖励。" palette={palette} />
            </div>
          ) : null}
        </section>
      </div>

      {rewardPopup ? <RewardOverlay popup={rewardPopup} palette={palette} /> : null}
    </div>
  );
}

function MetricBlock({ label, value, fullValue, suffix, note, palette, strong = false }) {
  const amountParts = strong ? splitAmountParts(String(value)) : null;
  return (
    <div title={fullValue || value} style={{ ...metricBlockStyle, background: palette.metricBackground, boxShadow: palette.metricShadow }}>
      <div style={{ fontSize: 14, color: palette.textSecondary }}>{label}</div>
      <div style={{ marginTop: 8, display: "flex", alignItems: "baseline", gap: 6 }}>
        <span style={{ ...metricValueStyle, fontSize: strong ? "clamp(30px, 4vw, 42px)" : "clamp(30px, 3.6vw, 38px)" }}>
          {strong && amountParts ? (
            <span style={amountValueWrapStyle}>
              <span>{amountParts.integer}</span>
              {amountParts.decimal ? <small style={amountDecimalStyle}>{amountParts.decimal}</small> : null}
              {amountParts.unit ? <small style={amountUnitStyle}>{amountParts.unit}</small> : null}
            </span>
          ) : value}
        </span>
        {suffix ? <span style={{ fontSize: 18, fontWeight: 700, color: palette.textSecondary }}>{suffix}</span> : null}
      </div>
      {note ? <div style={{ marginTop: 6, fontSize: 13, color: palette.textMuted }}>{note}</div> : null}
    </div>
  );
}

function RuleItem({ icon, title, text, palette }) {
  return (
    <div style={{ borderRadius: 8, padding: 18, background: palette.panelGlass, boxShadow: palette.listShadow }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, fontWeight: 700, marginBottom: 8 }}>
        <span style={{ ...ruleIconStyle, background: palette.infoSoft, color: palette.infoText }}>
          <IconMask name={icon} size={16} />
        </span>
        {title}
      </div>
      <div style={{ fontSize: 14, lineHeight: 1.6, color: palette.textSecondary }}>{text}</div>
    </div>
  );
}

function IconMask({ name, size = 16 }) {
  return (
    <span
      aria-hidden="true"
      style={{
        display: "inline-block",
        width: size,
        height: size,
        background: "currentColor",
        WebkitMask: `url("${assetUrl(name)}") center / contain no-repeat`,
        mask: `url("${assetUrl(name)}") center / contain no-repeat`,
      }}
    />
  );
}

function assetUrl(name) {
  return `${ASSET_BASE_URL}${name}?v=${ASSET_VERSION}`;
}

function EmptyState({ text, palette }) {
  return (
    <div style={{ border: `1px dashed ${palette.border}`, borderRadius: 8, padding: 24, color: palette.textSecondary }}>
      {text}
    </div>
  );
}

function RewardOverlay({ popup, palette }) {
  return (
    <div style={overlayStyle}>
      <div style={{ ...overlayCardStyle, background: palette.panelBackground, borderColor: palette.border }}>
        <div style={{ fontSize: 16, color: palette.textSecondary, marginBottom: 8 }}>签到成功</div>
        <div style={{ fontSize: 56, fontWeight: 800, color: palette.buttonPrimary, animation: "popReward 0.7s ease-out" }}>
          +{popup.totalReward}
        </div>
        <div style={{ fontSize: 15, color: palette.textSecondary }}>当前余额 {popup.balance}</div>
        <div style={{ marginTop: 8, fontSize: 14, color: palette.successText }}>连续签到 {popup.streak} 天</div>
        <div style={particleLayerStyle}>
          {Array.from({ length: 10 }, (_, index) => (
            <span key={index} style={{ ...particleStyle(index), background: palette.buttonPrimary }} />
          ))}
        </div>
      </div>
    </div>
  );
}

function particleStyle(index) {
  return {
    position: "absolute",
    left: `${12 + index * 8}%`,
    bottom: 18,
    width: 8,
    height: 8,
    borderRadius: "50%",
    opacity: 0.72,
    animation: `floatParticle 1.4s ease-out ${index * 0.04}s forwards`,
  };
}

function getTodayKey() {
  const date = new Date();
  return formatDateKey(date);
}

function buildCurrentMonthDays() {
  const now = new Date();
  const year = now.getFullYear();
  const month = now.getMonth();
  const lastDay = new Date(year, month + 1, 0).getDate();

  return Array.from({ length: lastDay }, (_, index) => {
    const day = index + 1;
    const date = new Date(year, month, day);
    return {
      day,
      dateKey: formatDateKey(date),
    };
  });
}

function formatDateKey(date) {
  const yyyy = date.getFullYear();
  const mm = String(date.getMonth() + 1).padStart(2, "0");
  const dd = String(date.getDate()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}`;
}

function formatStatus(status) {
  switch (status) {
    case "success":
      return "已到账";
    case "pending":
      return "待发放";
    case "failed":
      return "待重试";
    default:
      return status || "未知";
  }
}

function formatDateTime(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).format(date).replace(/\//g, "-");
}

function formatTimeOnly(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
}

function formatAmount(value, options = {}) {
  const numberValue = Number(value);
  if (!Number.isFinite(numberValue)) {
    return value;
  }

  if (options.compact) {
    const absValue = Math.abs(numberValue);
    if (absValue >= 100000000) {
      return `${trimFixed(numberValue / 100000000)} 亿`;
    }
    if (absValue >= 10000) {
      return `${trimFixed(numberValue / 10000)} 万`;
    }
  }

  return new Intl.NumberFormat("zh-CN", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(numberValue);
}

function splitAmountParts(value) {
  const match = value.match(/^([^.,\s]+(?:[,\d]*))(?:([.]\d{1,2}))?(?:\s*(.+))?$/);
  if (!match) {
    return null;
  }

  return {
    integer: match[1],
    decimal: match[2] || "",
    unit: match[3] || "",
  };
}

function trimFixed(value) {
  return value.toLocaleString("zh-CN", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  });
}

function getPalette(darkMode) {
  if (darkMode) {
    return {
      pageBackground: "#08101b",
      panelBackground: "#0f1b2d",
      panelGlass: "rgba(15,27,45,0.84)",
      mainCardBackground: "linear-gradient(145deg, rgba(33,44,61,0.92), rgba(15,27,45,0.86) 48%, rgba(60,42,28,0.52))",
      ambientBackground: "linear-gradient(135deg, rgba(246,196,83,0.10), rgba(20,184,166,0.12) 48%, rgba(59,130,246,0.07))",
      border: "#24334a",
      softBorder: "rgba(148,163,184,0.20)",
      textPrimary: "#edf4ff",
      textSecondary: "#9fb3cf",
      textMuted: "#6f8098",
      buttonPrimary: "#14b8a6",
      buttonDisabled: "#153746",
      successButton: "#123f3a",
      successButtonText: "#96f0df",
      buttonGlow: "0 10px 24px rgba(20,184,166,0.24)",
      cardShadow: "0 26px 76px rgba(0,0,0,0.24)",
      metricBackground: "rgba(255,255,255,0.03)",
      metricShadow: "0 12px 32px rgba(0,0,0,0.12)",
      todayGlow: "0 0 0 4px rgba(20,184,166,0.12), 0 10px 26px rgba(20,184,166,0.16)",
      rewardText: "#f6c453",
      rewardSoft: "rgba(246,196,83,0.08)",
      rewardBorder: "rgba(246,196,83,0.18)",
      rewardGlow: "0 10px 26px rgba(246,196,83,0.22)",
      infoSoft: "rgba(255,255,255,0.035)",
      infoText: "#68e0d2",
      listShadow: "0 14px 36px rgba(0,0,0,0.12)",
      progressLineActive: "linear-gradient(90deg, rgba(20,184,166,0.70), rgba(20,184,166,0.22))",
      progressNodeActive: "linear-gradient(145deg, rgba(20,184,166,0.42), rgba(246,196,83,0.16))",
      progressNodeIdle: "rgba(255,255,255,0.04)",
      monthDayIdle: "rgba(255,255,255,0.035)",
      successSoft: "rgba(20,184,166,0.12)",
      successText: "#68e0d2",
      alertBackground: "rgba(239,68,68,0.12)",
      alertText: "#fca5a5",
      alertBorder: "rgba(239,68,68,0.36)",
    };
  }

  return {
    pageBackground: "#f4fbfb",
    panelBackground: "#ffffff",
    panelGlass: "rgba(255,255,255,0.92)",
    mainCardBackground: "linear-gradient(145deg, rgba(255,255,255,0.96), rgba(245,253,253,0.94) 48%, rgba(255,247,226,0.76))",
    ambientBackground: "linear-gradient(135deg, rgba(251,191,36,0.10), rgba(15,159,151,0.09) 50%, rgba(59,130,246,0.055))",
    border: "#dce9ec",
    softBorder: "rgba(148,163,184,0.22)",
    textPrimary: "#102131",
    textSecondary: "#5b6c7f",
    textMuted: "#8fa0ae",
    buttonPrimary: "#0f9f97",
    buttonDisabled: "#d7ebeb",
    successButton: "#dff7ef",
    successButtonText: "#087963",
    buttonGlow: "0 10px 24px rgba(15,159,151,0.18)",
    cardShadow: "0 24px 64px rgba(44, 86, 105, 0.15)",
    metricBackground: "rgba(247, 252, 253, 0.72)",
    metricShadow: "0 12px 28px rgba(41, 80, 96, 0.06)",
    todayGlow: "0 0 0 4px rgba(15,159,151,0.10), 0 10px 24px rgba(15,159,151,0.12)",
    rewardText: "#b7791f",
    rewardSoft: "rgba(251, 191, 36, 0.12)",
    rewardBorder: "rgba(183,121,31,0.16)",
    rewardGlow: "0 10px 24px rgba(183,121,31,0.18)",
    infoSoft: "rgba(240,248,249,0.70)",
    infoText: "#0f9f97",
    listShadow: "0 10px 28px rgba(44, 86, 105, 0.08)",
    progressLineActive: "linear-gradient(90deg, rgba(15,159,151,0.58), rgba(15,159,151,0.16))",
    progressNodeActive: "linear-gradient(145deg, rgba(219,250,241,0.96), rgba(255,246,219,0.92))",
    progressNodeIdle: "rgba(255,255,255,0.72)",
    monthDayIdle: "rgba(255,255,255,0.72)",
    successSoft: "#eafbf7",
    successText: "#0b7f6d",
    alertBackground: "#fff1f1",
    alertText: "#b42318",
    alertBorder: "#fecaca",
  };
}

const pageStyle = {
  minHeight: "100vh",
  width: "100%",
  position: "relative",
  overflowX: "hidden",
};

const pageInnerStyle = {
  maxWidth: 1180,
  margin: "0 auto",
  padding: "24px 20px 32px",
  position: "relative",
  zIndex: 1,
};

const ambientBandStyle = {
  position: "absolute",
  top: 0,
  left: 0,
  right: 0,
  height: "34vh",
  minHeight: 260,
  pointerEvents: "none",
};

const mainCardStyle = {
  position: "relative",
  overflow: "hidden",
  border: "1px solid",
  borderRadius: 24,
  padding: 0,
  backdropFilter: "blur(18px)",
};

const cardGlowStyle = {
  position: "absolute",
  top: -80,
  right: -60,
  width: 220,
  height: 220,
  borderRadius: "50%",
  background: "rgba(20,184,166,0.13)",
  filter: "blur(24px)",
  animation: "slowDrift 8s ease-in-out infinite",
};

const heroIllustrationStyle = {
  display: "flex",
  justifyContent: "flex-end",
  alignItems: "center",
  minWidth: 0,
  pointerEvents: "none",
};

const heroIllustrationShellStyle = {
  width: "min(300px, 100%)",
  minHeight: 198,
  display: "grid",
  placeItems: "center",
  borderRadius: 28,
  background: "linear-gradient(180deg, rgba(255,255,255,0.62), rgba(255,255,255,0.24))",
  border: "1px solid rgba(226, 232, 240, 0.26)",
  boxShadow: "0 18px 42px rgba(41, 80, 96, 0.06)",
  backdropFilter: "blur(14px)",
  padding: "14px 14px 8px",
};

const heroImageStyle = {
  width: "min(214px, 100%)",
  height: "auto",
  display: "block",
  filter: "saturate(0.88) contrast(0.97) opacity(0.88)",
};

const mainCardContentStyle = {
  position: "relative",
  zIndex: 1,
  padding: "34px 34px 30px",
};

const mainTopStyle = {
  display: "grid",
  gridTemplateColumns: "minmax(0, 1.15fr) minmax(260px, 0.85fr)",
  gap: 24,
  alignItems: "center",
};

const heroTextColumnStyle = {
  minWidth: 0,
};

const heroCopyStyle = {
  maxWidth: 680,
};

const eyebrowStyle = {
  fontSize: 14,
  fontWeight: 700,
};

const heroTitleStyle = {
  margin: "8px 0 12px",
  fontSize: "clamp(38px, 6vw, 54px)",
  lineHeight: 1.08,
  letterSpacing: 0,
};

const heroDescStyle = {
  fontSize: 16,
  lineHeight: 1.7,
  maxWidth: 620,
  margin: 0,
};

const rewardFocusStyle = {
  marginTop: 22,
  display: "grid",
  justifyItems: "start",
};

const rewardCenterStyle = {
  display: "grid",
  justifyItems: "center",
  gap: 4,
  animation: "rewardGlowIn 0.8s ease-out both",
};

const coinIconStyle = {
  width: 44,
  height: 44,
  border: "1px solid",
  borderRadius: "50%",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  fontSize: 22,
  fontWeight: 900,
};

const rewardCenterAmountStyle = {
  fontSize: "clamp(44px, 7vw, 70px)",
  lineHeight: 1,
  fontWeight: 900,
  letterSpacing: 0,
};

const rewardCenterStatusStyle = {
  marginTop: 8,
  borderRadius: 999,
  padding: "6px 12px",
  fontSize: 14,
  fontWeight: 800,
};

const pendingRewardStyle = {
  display: "flex",
  alignItems: "center",
  gap: 10,
  fontSize: 15,
};

const giftIconStyle = {
  width: 38,
  height: 38,
  borderRadius: "50%",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  fontSize: 15,
  fontWeight: 900,
};

const mainStatsStyle = {
  margin: "26px auto 0",
  display: "grid",
  gridTemplateColumns: "minmax(0, 1fr) minmax(0, 1.15fr)",
  gap: 14,
  alignItems: "stretch",
  maxWidth: 760,
};

const metricBlockStyle = {
  borderRadius: 16,
  padding: "18px 20px",
  minWidth: 0,
};

const metricValueStyle = {
  fontWeight: 850,
  letterSpacing: 0,
  lineHeight: 1.05,
  overflowWrap: "anywhere",
  fontVariantNumeric: "tabular-nums lining-nums",
  whiteSpace: "nowrap",
};

const amountValueWrapStyle = {
  display: "inline-flex",
  alignItems: "baseline",
  gap: 1,
  minWidth: 0,
  maxWidth: "100%",
};

const amountDecimalStyle = {
  fontSize: "0.62em",
  fontWeight: 800,
  opacity: 0.8,
};

const amountUnitStyle = {
  marginLeft: 4,
  fontSize: "0.5em",
  fontWeight: 800,
  opacity: 0.78,
};

const rewardHintStyle = {
  margin: "18px auto 0",
  display: "inline-flex",
  alignItems: "center",
  flexWrap: "wrap",
  gap: 6,
  borderRadius: 8,
  padding: "9px 11px",
  fontSize: 14,
};

const primaryActionRowStyle = {
  marginTop: 22,
  display: "flex",
  gap: 12,
  flexWrap: "wrap",
  alignItems: "center",
  justifyContent: "center",
};

const primaryButtonStyle = {
  border: "none",
  borderRadius: 8,
  padding: "15px 28px",
  fontSize: 16,
  fontWeight: 700,
  display: "inline-flex",
  alignItems: "center",
  justifyContent: "center",
  gap: 8,
  cursor: "pointer",
  transition: "transform 160ms ease, box-shadow 160ms ease, background 160ms ease",
};

const secondaryButtonStyle = {
  border: "1px solid",
  borderRadius: 8,
  padding: "10px 13px",
  background: "transparent",
  fontSize: 14,
  fontWeight: 600,
  cursor: "pointer",
  transition: "background 160ms ease, border-color 160ms ease",
};

const alertStyle = {
  marginTop: 16,
  border: "1px solid",
  borderRadius: 8,
  padding: "12px 14px",
};

const sectionStyle = {
  borderRadius: 8,
  padding: "24px 10px 0",
  background: "transparent",
};

const sectionTitleStyle = {
  margin: 0,
  fontSize: 24,
  letterSpacing: 0,
};

const sectionHeaderStyle = {
  display: "flex",
  justifyContent: "space-between",
  gap: 16,
  alignItems: "flex-end",
  marginBottom: 16,
};

const monthOverviewStyle = {
  display: "grid",
  gridTemplateColumns: "320px minmax(0, 1fr)",
  gap: 16,
  alignItems: "stretch",
};

const monthSummaryStyle = {
  borderRadius: 16,
  padding: 14,
  display: "grid",
  gap: 12,
};

const monthSummaryHeroStyle = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  gap: 14,
  borderRadius: 16,
  padding: "12px 14px",
  background: "linear-gradient(135deg, rgba(255,255,255,0.7), rgba(240,249,255,0.52))",
  border: "1px solid rgba(148, 163, 184, 0.12)",
};

const monthSummaryImageStyle = {
  width: 72,
  height: 72,
  flexShrink: 0,
  objectFit: "contain",
  filter: "drop-shadow(0 10px 24px rgba(15, 23, 42, 0.08))",
};

const monthCalendarStyle = {
  borderRadius: 16,
  padding: 16,
  display: "grid",
  gridTemplateColumns: "repeat(7, minmax(34px, 1fr))",
  gap: 8,
};

const monthDayStyle = {
  position: "relative",
  minHeight: 36,
  border: "1px solid",
  borderRadius: 10,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  fontSize: 13,
  fontWeight: 800,
};

const monthDayMarkStyle = {
  position: "absolute",
  right: 4,
  bottom: 2,
  fontSize: 10,
  lineHeight: 1,
};

const sectionDescStyle = {
  margin: "8px 0 0",
  fontSize: 14,
  lineHeight: 1.6,
};

const progressSectionStyle = {
  marginTop: 24,
  borderTop: "1px solid",
  padding: "24px 10px 0",
};

const timelineGridStyle = {
  display: "grid",
  gridTemplateColumns: "repeat(7, minmax(0, 1fr))",
  gap: 10,
  position: "relative",
};

const timelineItemStyle = {
  position: "relative",
  minWidth: 0,
  display: "grid",
  justifyItems: "center",
  gap: 8,
};

const timelineDotStyle = {
  border: "2px solid",
  borderRadius: "50%",
  width: "clamp(48px, 7vw, 64px)",
  height: "clamp(48px, 7vw, 64px)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  transition: "background 200ms ease, box-shadow 200ms ease, border-color 200ms ease",
};

const timelineDateStyle = {
  fontSize: 13,
  lineHeight: 1.2,
  textAlign: "center",
  fontWeight: 700,
};

const timelineLabelStyle = {
  fontSize: 13,
  fontWeight: 700,
  textAlign: "center",
};

const timelineLineStyle = {
  position: "absolute",
  top: "31%",
  left: "calc(50% + 28px)",
  width: "calc(100% - 46px)",
  height: 6,
  borderRadius: 999,
  zIndex: -1,
  transformOrigin: "left center",
  transition: "background 220ms ease, opacity 220ms ease, transform 220ms ease",
};

const ledgerItemStyle = {
  border: "none",
  borderRadius: 16,
  padding: "17px 20px",
  display: "grid",
  gridTemplateColumns: "minmax(150px, 1fr) minmax(160px, 1fr) auto",
  alignItems: "center",
  gap: 18,
};

const ledgerLeadStyle = {
  display: "flex",
  alignItems: "center",
  gap: 14,
  minWidth: 0,
};

const ledgerIconStyle = {
  width: 44,
  height: 44,
  borderRadius: 14,
  display: "inline-flex",
  alignItems: "center",
  justifyContent: "center",
  flexShrink: 0,
  boxShadow: "inset 0 1px 0 rgba(255,255,255,0.5)",
};

const ledgerTitleStyle = {
  fontSize: 18,
  fontWeight: 800,
  letterSpacing: 0,
};

const ledgerSplitStyle = {
  fontSize: 14,
  lineHeight: 1.6,
  display: "grid",
  gap: 8,
};

const ledgerSummaryTextStyle = {
  fontSize: 13,
  lineHeight: 1.6,
};

const ledgerSplitRowStyle = {
  display: "flex",
  gap: 8,
  flexWrap: "wrap",
};

const splitTagStyle = {
  display: "inline-flex",
  width: "fit-content",
  borderRadius: 999,
  padding: "4px 9px",
  fontSize: 13,
};

const ledgerAmountWrapStyle = {
  textAlign: "right",
  display: "grid",
  justifyItems: "end",
  gap: 3,
};

const statusPillStyle = {
  borderRadius: 8,
  padding: "4px 8px",
  fontSize: 12,
  fontWeight: 700,
};

const rulesSectionStyle = {
  marginTop: 24,
  borderTop: "1px solid",
  padding: "18px 10px 0",
};

const rulesToggleStyle = {
  width: "100%",
  border: "none",
  background: "transparent",
  padding: 0,
  display: "flex",
  justifyContent: "space-between",
  gap: 16,
  alignItems: "center",
  fontSize: 18,
  fontWeight: 800,
  cursor: "pointer",
};

const rulesSummaryStyle = {
  marginTop: 8,
  fontSize: 14,
};

const rulesActionStyle = {
  border: "1px solid",
  borderRadius: 8,
  padding: "6px 10px",
  fontSize: 13,
  fontWeight: 700,
};

const ruleIconStyle = {
  width: 28,
  height: 28,
  borderRadius: 8,
  display: "inline-flex",
  alignItems: "center",
  justifyContent: "center",
  fontSize: 13,
  fontWeight: 900,
};

const rulesGridStyle = {
  marginTop: 16,
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
  gap: 12,
};

const overlayStyle = {
  position: "fixed",
  inset: 0,
  background: "rgba(2, 10, 24, 0.36)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  zIndex: 20,
};

const overlayCardStyle = {
  position: "relative",
  overflow: "hidden",
  minWidth: 320,
  border: "1px solid",
  borderRadius: 8,
  padding: "28px 24px",
  textAlign: "center",
};

const particleLayerStyle = {
  position: "absolute",
  inset: 0,
  pointerEvents: "none",
};

const animationStyles = `
  @keyframes buttonAura {
    0%, 100% { transform: translateY(0) scale(1); filter: brightness(1); }
    50% { transform: translateY(-1px) scale(1.02); filter: brightness(1.04); }
  }

  @keyframes slowDrift {
    0%, 100% { transform: translate3d(0, 0, 0); opacity: 0.78; }
    50% { transform: translate3d(-18px, 18px, 0); opacity: 1; }
  }

  @keyframes rewardGlowIn {
    0% { transform: translateY(8px) scale(0.92); opacity: 0; }
    70% { transform: translateY(0) scale(1.04); opacity: 1; }
    100% { transform: translateY(0) scale(1); opacity: 1; }
  }

  @keyframes timelinePulse {
    0% { transform: translateY(0) scale(0.88); opacity: 0.72; }
    65% { transform: translateY(-2px) scale(1.05); opacity: 1; }
    100% { transform: translateY(-2px) scale(1); opacity: 1; }
  }

  @keyframes todayHalo {
    0%, 100% { box-shadow: 0 0 0 0 rgba(45, 212, 191, 0.14); }
    50% { box-shadow: 0 0 0 8px rgba(45, 212, 191, 0.06); }
  }

  @keyframes popReward {
    0% { transform: scale(0.72); opacity: 0; }
    65% { transform: scale(1.08); opacity: 1; }
    100% { transform: scale(1); opacity: 1; }
  }

  @keyframes floatParticle {
    0% { transform: translateY(0) scale(0.6); opacity: 0.8; }
    100% { transform: translateY(-140px) scale(1.1); opacity: 0; }
  }

  @media (max-width: 760px) {
    h1 { font-size: 34px !important; }
    button { min-height: 48px; }
    .mobile-stack { grid-template-columns: 1fr !important; }
    .ledger-row { justify-items: stretch !important; }
    .hero-illustration { justify-content: center !important; opacity: 1 !important; }
  }
`;

ReactDOM.createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
