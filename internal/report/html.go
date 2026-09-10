package report

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/johan-larp/agentsearch/internal/models"
)

// HTMLReport renders a self-contained interactive report.
type HTMLReport struct{}

func NewHTMLReport() *HTMLReport {
	return &HTMLReport{}
}

// Parse the template once so invalid templates fail at startup.
var reportTmpl = template.Must(template.New("report").Parse(htmlTemplate))

func (h *HTMLReport) Generate(target string, results []models.Result, duration time.Duration) (string, error) {
	target, results = normalizeResults(target, results)
	summary := BuildSummary(target, results, duration)
	path := filepath.Join("output", fmt.Sprintf("%s_report.html", sanitizeFilename(target)))

	if err := os.MkdirAll("output", 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	data := struct {
		Target    string
		Duration  string
		Summary   Summary
		Results   []models.Result
		Timestamp string
	}{
		Target:    target,
		Duration:  duration.Round(time.Second).String(),
		Summary:   summary,
		Results:   results,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}

	if err := reportTmpl.Execute(f, data); err != nil {
		return "", err
	}
	return path, nil
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en" data-report="agentsearch">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light">
<meta name="robots" content="noindex, nofollow">
<title>AgentSearch — {{.Target}}</title>
<style>
/* ============================================================
   AgentSearch · report template
   White, minimal, cinematic. Cards laid out in a 3-column grid.
   ============================================================ */
:root{
  --bg:#ffffff;
  --ink:#0b0b0c;
  --ink-2:#4b5563;
  --ink-3:#9ca3af;
  --line:#e6e6e8;
  --line-strong:#d6d6da;

  --found:#10b981;
  --blocked:#f59e0b;
  --notfound:#9ca3af;
  --error:#ef4444;
  --link:#3b82f6;

  --shadow-rest:0 1px 2px rgba(0,0,0,.04), 0 0 1px rgba(0,0,0,.10);
  --shadow-hover:0 10px 40px rgba(0,0,0,.15), 0 0 1px rgba(0,0,0,.10);
  --shadow-open:0 18px 60px rgba(0,0,0,.16), 0 0 1px rgba(0,0,0,.10);

  --r:14px;
  --pad:1.5rem;
  --ease:cubic-bezier(.25,.46,.45,.94);
  --maxw:1240px;
}

*,*::before,*::after{box-sizing:border-box}
html{-webkit-text-size-adjust:100%}
body{
  margin:0;
  background:var(--bg);
  color:var(--ink);
  font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Inter,Roboto,"Helvetica Neue",Arial,sans-serif;
  font-size:16px;
  line-height:1.5;
  -webkit-font-smoothing:antialiased;
  text-rendering:optimizeLegibility;
}
a{color:var(--link);text-decoration:none}
a:hover{text-decoration:underline}
:focus-visible{outline:2px solid var(--link);outline-offset:3px;border-radius:6px}

.skip{position:absolute;left:-9999px;top:0;background:#fff;padding:.75rem 1rem;border:1px solid var(--line);border-radius:8px;z-index:99}
.skip:focus{left:1rem;top:1rem}

.wrap{max-width:var(--maxw);margin:0 auto;padding:clamp(2.5rem,6vw,5.5rem) clamp(1rem,4vw,2.5rem) 5rem}

/* ---------- header ---------- */
.masthead{text-align:center;margin-bottom:clamp(2rem,5vw,3.5rem)}
.masthead h1{
  margin:0;
  font-family:"Iowan Old Style","Palatino Linotype",Palatino,Georgia,"Times New Roman",serif;
  font-weight:600;
  font-size:clamp(2.6rem,8vw,5.25rem);
  letter-spacing:.16em;
  text-transform:uppercase;
  line-height:1.02;
  text-indent:.16em;
}
.masthead .sub{
  display:block;margin-top:.35rem;
  font-size:clamp(.95rem,2.2vw,1.25rem);
  letter-spacing:.42em;text-indent:.42em;
  color:var(--ink-3);font-weight:400;
}
.masthead .meta{
  margin:1.75rem auto 0;
  display:flex;flex-wrap:wrap;gap:.5rem .75rem;justify-content:center;
  font-size:.8125rem;color:var(--ink-2);
}
.masthead .meta span{
  border:1px solid var(--line);border-radius:999px;
  padding:.32rem .8rem;background:#fff;
}
.masthead .meta b{font-weight:600;color:var(--ink)}
.rule{height:1px;background:linear-gradient(90deg,transparent,var(--line-strong) 18%,var(--line-strong) 82%,transparent);margin:clamp(2rem,5vw,3rem) 0}

/* ---------- stats · monochrome, no accents ---------- */
.stats{
  display:grid;
  grid-template-columns:repeat(5,minmax(0,1fr));
  gap:clamp(1rem,3vw,2.5rem) 1rem;
}
@media (max-width:900px){ .stats{grid-template-columns:repeat(3,minmax(0,1fr))} }
@media (max-width:520px){ .stats{grid-template-columns:repeat(2,minmax(0,1fr))} }
.stat{padding:0;background:none;border:0;box-shadow:none}
.stat .n{
  display:block;
  font-size:clamp(2.1rem,4.6vw,3rem);
  font-weight:400;letter-spacing:-.03em;line-height:1;
  font-variant-numeric:tabular-nums;
  color:var(--ink);
}
.stat .l{
  display:block;margin-top:.6rem;
  font-size:.7rem;letter-spacing:.16em;text-transform:uppercase;
  color:var(--ink-3);font-weight:500;
}
.stat--muted .n{color:var(--ink-3)}

/* ---------- toolbar ---------- */
.toolbar{margin:clamp(1.75rem,4vw,2.5rem) 0 1.5rem;display:grid;gap:1rem}
.search{position:relative}
.search svg{position:absolute;left:1.05rem;top:50%;transform:translateY(-50%);width:18px;height:18px;stroke:var(--ink-3);fill:none;stroke-width:1.7;pointer-events:none}
.search input{
  width:100%;font:inherit;color:var(--ink);
  padding:1rem 3rem 1rem 3rem;
  border:1px solid var(--line);border-radius:var(--r);background:#fff;
  box-shadow:var(--shadow-rest);
  transition:border-color .3s var(--ease),box-shadow .3s var(--ease),transform .3s var(--ease);
}
.search input::placeholder{color:var(--ink-3)}
.search input:focus{outline:none;border-color:#c9c9cf;box-shadow:0 10px 40px rgba(0,0,0,.10),0 0 0 3px rgba(59,130,246,.14)}
.search .clear{
  position:absolute;right:.55rem;top:50%;transform:translateY(-50%);
  border:0;background:transparent;color:var(--ink-3);cursor:pointer;
  font-size:1.1rem;line-height:1;padding:.5rem .65rem;border-radius:8px;
}
.search .clear:hover{color:var(--ink);background:#f4f4f5}
.chips{display:flex;flex-wrap:wrap;gap:.5rem;align-items:center}
.chip{
  font:inherit;font-size:.8125rem;cursor:pointer;
  border:1px solid var(--line);background:#fff;color:var(--ink-2);
  padding:.45rem .9rem;border-radius:999px;
  transition:transform .3s var(--ease),box-shadow .3s var(--ease),color .3s var(--ease),border-color .3s var(--ease);
}
.chip:hover{transform:translateY(-1px);box-shadow:var(--shadow-rest);color:var(--ink)}
.chip[aria-pressed="true"]{background:var(--ink);border-color:var(--ink);color:#fff}
.chip .dot{display:inline-block;width:7px;height:7px;border-radius:50%;margin-right:.45rem;vertical-align:middle;background:var(--tone,var(--ink-3))}
.chip--found{--tone:var(--found)} .chip--blocked{--tone:var(--blocked)}
.chip--not_found{--tone:var(--notfound)} .chip--error{--tone:var(--error)}
.count{margin-left:auto;font-size:.8125rem;color:var(--ink-3);font-variant-numeric:tabular-nums}

/* ---------- grid: three per row ---------- */
.grid{
  display:grid;
  grid-template-columns:repeat(3,minmax(0,1fr));
  gap:1rem;
  align-items:start;
  list-style:none;margin:0;padding:0;
}
@media (max-width:1199px){ .grid{grid-template-columns:repeat(2,minmax(0,1fr))} }
@media (max-width:767px){ .grid{grid-template-columns:1fr} }

/* ---------- card ---------- */
.card{
  border:1px solid var(--line);border-radius:var(--r);background:#fff;
  box-shadow:var(--shadow-rest);
  transition:transform .45s var(--ease),box-shadow .45s var(--ease),border-color .45s var(--ease);
  contain:layout paint style;
}
.card:hover{transform:translateY(-4px);box-shadow:var(--shadow-hover)}
.card.is-open{box-shadow:var(--shadow-open);border-color:var(--line-strong)}
.card.is-hidden{display:none}

/* staggered entry — transform + opacity only */
.card,.stat{will-change:transform,opacity}
.js .card{opacity:0;transform:translate3d(0,-14px,0)}
.js .card.is-in{
  animation:slide-in .62s var(--ease) both;
  animation-delay:calc(var(--i,0) * 55ms);
}
@keyframes slide-in{from{opacity:0;transform:translate3d(0,-14px,0)}to{opacity:1;transform:translate3d(0,0,0)}}
.card.is-in{will-change:auto}

.card__head{
  all:unset;
  display:flex;align-items:center;gap:1rem;
  width:100%;box-sizing:border-box;
  padding:var(--pad);cursor:pointer;
}
.card__head:focus-visible{outline:2px solid var(--link);outline-offset:-4px;border-radius:var(--r)}
.card__name{
  font-weight:600;font-size:1.0625rem;letter-spacing:-.01em;
  overflow:hidden;text-overflow:ellipsis;white-space:nowrap;flex:1 1 auto;min-width:0;
}
.badge{
  flex:0 0 auto;display:inline-flex;align-items:center;gap:.4rem;
  font-size:.72rem;letter-spacing:.08em;text-transform:uppercase;font-weight:600;
  color:var(--tone);background:#fafafa;border:1px solid var(--line);
  padding:.32rem .6rem;border-radius:999px;white-space:nowrap;
}
@supports (background:color-mix(in srgb,red 10%,#fff)){
  .badge{
    background:color-mix(in srgb,var(--tone) 10%,#fff);
    border-color:color-mix(in srgb,var(--tone) 30%,#fff);
  }
}
.badge .dot{width:6px;height:6px;border-radius:50%;background:var(--tone)}
.card--found{--tone:var(--found)}
.card--blocked{--tone:var(--blocked)}
.card--not_found{--tone:var(--notfound)}
.card--error{--tone:var(--error)}
.chev{flex:0 0 auto;width:14px;height:14px;stroke:var(--ink-3);fill:none;stroke-width:2;transition:transform .45s var(--ease)}
.card.is-open .chev{transform:rotate(180deg)}

.card__body{
  max-height:0;overflow:hidden;
  transition:max-height .5s var(--ease),opacity .35s var(--ease);
  opacity:0;
}
.card.is-open .card__body{opacity:1}
.card__inner{padding:0 var(--pad) var(--pad);border-top:1px solid var(--line);margin:0 var(--pad);padding-left:0;padding-right:0}
.card__inner > *:first-child{margin-top:1.1rem}

.kv{display:grid;grid-template-columns:auto 1fr;gap:.5rem .9rem;font-size:.875rem;margin:0}
.kv dt{color:var(--ink-3);font-size:.72rem;letter-spacing:.1em;text-transform:uppercase;padding-top:.15rem}
.kv dd{margin:0;color:var(--ink-2);word-break:break-word;overflow-wrap:anywhere}
.kv dd .url{color:var(--link)}
.kv dd b{color:var(--ink);font-weight:600}

.conf{margin-top:1.1rem}
.conf__top{display:flex;justify-content:space-between;align-items:baseline;font-size:.72rem;letter-spacing:.1em;text-transform:uppercase;color:var(--ink-3)}
.conf__val{font-size:1.05rem;font-weight:600;color:var(--ink);letter-spacing:-.01em;font-variant-numeric:tabular-nums}
.bar{margin-top:.5rem;height:4px;border-radius:999px;background:#f0f0f2;overflow:hidden}
.bar > i{display:block;height:100%;width:0;border-radius:999px;background:var(--tone,var(--found));transform-origin:left center;transition:width .8s var(--ease)}
.card.is-open .bar > i{width:var(--w,0%)}

.err{margin-top:1.1rem;font-size:.85rem;color:var(--error);background:rgba(239,68,68,.06);border:1px solid rgba(239,68,68,.2);border-radius:10px;padding:.7rem .85rem;overflow-wrap:anywhere}

.actions{display:flex;flex-wrap:wrap;gap:.6rem;margin-top:1.25rem}
.btn{
  font:inherit;font-size:.8125rem;cursor:pointer;text-decoration:none;
  border:1px solid var(--line);background:#fff;color:var(--ink-2);
  padding:.5rem .9rem;border-radius:10px;
  transition:transform .3s var(--ease),box-shadow .3s var(--ease),color .3s var(--ease);
}
.btn:hover{transform:translateY(-1px);box-shadow:var(--shadow-rest);color:var(--ink);text-decoration:none}
.btn--primary{border-color:rgba(59,130,246,.35);color:var(--link)}

.empty{
  display:none;text-align:center;padding:4rem 1rem;color:var(--ink-3);
  border:1px dashed var(--line-strong);border-radius:var(--r);
}
.empty.is-on{display:block}
.foot{margin-top:4rem;text-align:center;font-size:.75rem;color:var(--ink-3);letter-spacing:.06em}
.sr{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0 0 0 0);white-space:nowrap;border:0}

@media (prefers-reduced-motion:reduce){
  *,*::before,*::after{animation-duration:.001ms !important;animation-delay:0ms !important;transition-duration:.001ms !important}
  .js .card{opacity:1;transform:none}
}
/* no-JS fallback: everything expanded, filters hidden */
html:not(.js) .card__body{max-height:none;opacity:1}
html:not(.js) .card__body[hidden]{display:block}
html:not(.js) .toolbar,html:not(.js) .chev{display:none}

@media print{
  .toolbar,.chev,.foot{display:none}
  .card{break-inside:avoid;box-shadow:none}
  .card__body{max-height:none !important;opacity:1 !important}
  .grid{grid-template-columns:repeat(2,1fr)}
}
</style>
</head>
<body>
<a class="skip" href="#results">Skip to results</a>

<div class="wrap">

  <header class="masthead">
    <h1>Agent Search<span class="sub">results</span></h1>
    <div class="meta">
      <span>target&nbsp;·&nbsp;<b>{{.Target}}</b></span>
      <span>elapsed&nbsp;·&nbsp;<b>{{.Duration}}</b></span>
      <span>generated&nbsp;·&nbsp;<b>{{.Timestamp}}</b></span>
    </div>
  </header>

  <!-- ================= statistics ================= -->
  <section class="stats" aria-label="Summary statistics">
    <div class="stat"><span class="n">{{.Summary.Total}}</span><span class="l">Total checks</span></div>
    <div class="stat"><span class="n">{{.Summary.Found}}</span><span class="l">Found</span></div>
    <div class="stat stat--muted"><span class="n">{{.Summary.NotFound}}</span><span class="l">Not found</span></div>
    <div class="stat stat--muted"><span class="n">{{.Summary.Blocked}}</span><span class="l">Blocked</span></div>
    <div class="stat stat--muted"><span class="n">{{.Summary.Errors}}</span><span class="l">Errors</span></div>
  </section>

  <div class="rule"></div>

  <!-- ================= toolbar ================= -->
  <div class="toolbar">
    <div class="search">
      <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7"/><path d="M20 20l-3.5-3.5"/></svg>
      <label class="sr" for="q">Filter results by source name</label>
      <input id="q" type="search" autocomplete="off" spellcheck="false"
             placeholder="Filter results by source name..." aria-controls="results">
      <button class="clear" type="button" id="clear" title="Clear filter" aria-label="Clear filter" hidden>&times;</button>
    </div>
    <div class="chips" role="group" aria-label="Filter by status">
      <button class="chip" type="button" data-status="all" aria-pressed="true">All</button>
      <button class="chip chip--found" type="button" data-status="found" aria-pressed="false"><span class="dot"></span>Found</button>
      <button class="chip chip--blocked" type="button" data-status="blocked" aria-pressed="false"><span class="dot"></span>Blocked</button>
      <button class="chip chip--not_found" type="button" data-status="not_found" aria-pressed="false"><span class="dot"></span>Not found</button>
      <button class="chip chip--error" type="button" data-status="error" aria-pressed="false"><span class="dot"></span>Errors</button>
      <span class="count" id="count" role="status" aria-live="polite"></span>
    </div>
  </div>

  <!-- ================= results (3 per row) ================= -->
  <ul class="grid" id="results">
    {{range .Results}}
    <li class="card card--{{.Status}}"
        data-name="{{.SiteName}}"
        data-status="{{.Status}}"
        data-confidence="{{.Confidence}}">
      <button class="card__head" type="button" aria-expanded="false">
        <span class="card__name" title="{{.SiteName}}">{{.SiteName}}</span>
        {{if eq .Status "found"}}<span class="badge"><span class="dot"></span>Found</span>
        {{else if eq .Status "blocked"}}<span class="badge"><span class="dot"></span>Blocked</span>
        {{else if eq .Status "error"}}<span class="badge"><span class="dot"></span>Error</span>
        {{else}}<span class="badge"><span class="dot"></span>Not found</span>{{end}}
        <svg class="chev" viewBox="0 0 24 24" aria-hidden="true"><path d="M6 9l6 6 6-6"/></svg>
      </button>

      <div class="card__body" role="region" hidden>
        <div class="card__inner">
          <dl class="kv">
            <dt>URL</dt>
            <dd><a class="url" href="{{.URL}}" target="_blank" rel="noopener noreferrer nofollow">{{.URL}}</a></dd>
            {{if .FinalURL}}
            <dt>Final</dt>
            <dd><a class="url" href="{{.FinalURL}}" target="_blank" rel="noopener noreferrer nofollow">{{.FinalURL}}</a></dd>
            {{end}}
            <dt>Status</dt>
            <dd><b>{{if eq .Status "found"}}Found{{else if eq .Status "blocked"}}Blocked{{else if eq .Status "error"}}Error{{else}}Not found{{end}}</b></dd>
            <dt>Latency</dt>
            <dd>{{.Duration}}</dd>
          </dl>

          {{if eq .Status "found"}}
          <div class="conf">
            <div class="conf__top"><span>Confidence</span><span class="conf__val">{{.Confidence}}%</span></div>
            <div class="bar" role="img" aria-label="Confidence {{.Confidence}} percent"><i style="--w:{{.Confidence}}%"></i></div>
          </div>
          {{end}}

          {{if .Error}}<p class="err">{{.Error}}</p>{{end}}
          {{if or .Evidence .Metadata}}<dl class="kv">
            {{range .Evidence}}<dt>{{.Kind}}</dt><dd>{{.Value}}</dd>{{end}}
            {{range $key, $value := .Metadata}}<dt>{{$key}}</dt><dd>{{$value}}</dd>{{end}}
          </dl>{{end}}

          <div class="actions">
            <a class="btn btn--primary" href="{{.URL}}" target="_blank" rel="noopener noreferrer nofollow">Open source ↗</a>
            <button class="btn" type="button" data-copy="{{.URL}}">Copy URL</button>
            <button class="btn" type="button" data-collapse>Close</button>
          </div>
        </div>
      </div>
    </li>
    {{end}}
  </ul>

  <p class="empty" id="empty">No sources match this filter.</p>

  <footer class="foot">AgentSearch · {{.Target}} · {{.Timestamp}}</footer>
</div>

<script>
(function () {
  "use strict";
  document.documentElement.classList.add("js");

  var grid  = document.getElementById("results");
  if (!grid) return;
  var cards = Array.prototype.slice.call(grid.children);
  var input = document.getElementById("q");
  var clear = document.getElementById("clear");
  var empty = document.getElementById("empty");
  var count = document.getElementById("count");
  var chips = Array.prototype.slice.call(document.querySelectorAll(".chip"));

  /* ---------- custom popularity sort (not alphabetical) ---------- */
  var TIERS = [
    ["vk","vkontakte","x","x (twitter)","twitter","telegram","facebook","instagram","tiktok"],
    ["linkedin","github","gitlab"],
    ["reddit","discord","medium"]
  ];
  var rank = Object.create(null);
  TIERS.forEach(function (list, tier) {
    list.forEach(function (name, idx) { rank[name] = tier * 1000 + idx; });
  });
  function key(el) {
    var n = (el.dataset.name || "").trim().toLowerCase();
    if (n in rank) return rank[n];
    var bare = n.replace(/\s*\(.*?\)\s*/g, "").replace(/\.(com|org|net|io|ru)$/, "").trim();
    if (bare in rank) return rank[bare];
    return 3000; /* fourth tier — sorted alphabetically below */
  }
  cards.sort(function (a, b) {
    var ka = key(a), kb = key(b);
    if (ka !== kb) return ka - kb;
    return (a.dataset.name || "").localeCompare(b.dataset.name || "", undefined, { sensitivity: "base" });
  });
  var frag = document.createDocumentFragment();
  cards.forEach(function (c) { frag.appendChild(c); });
  grid.appendChild(frag);

  /* ---------- staggered entry, batched for large reports ---------- */
  var BATCH = 60;
  function reveal(list) {
    list.forEach(function (c, i) {
      c.style.setProperty("--i", i < 40 ? i : 40);
      c.classList.add("is-in");
    });
  }
  if ("IntersectionObserver" in window && cards.length > BATCH) {
    reveal(cards.slice(0, BATCH));
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) {
        if (!e.isIntersecting) return;
        e.target.style.setProperty("--i", 0);
        e.target.classList.add("is-in");
        io.unobserve(e.target);
      });
    }, { rootMargin: "300px 0px" });
    cards.slice(BATCH).forEach(function (c) { io.observe(c); });
  } else {
    reveal(cards);
  }

  /* ---------- expand / collapse ---------- */
  var uid = 0;
  cards.forEach(function (card) {
    var head = card.querySelector(".card__head");
    var body = card.querySelector(".card__body");
    if (!head || !body) return;
    var id = "panel-" + (++uid);
    body.id = id;
    head.setAttribute("aria-controls", id);
    body.setAttribute("aria-label", (card.dataset.name || "result") + " details");

    function setOpen(open) {
      card.classList.toggle("is-open", open);
      head.setAttribute("aria-expanded", open ? "true" : "false");
      if (open) {
        body.hidden = false;
        body.style.maxHeight = body.scrollHeight + "px";
      } else {
        body.style.maxHeight = "0px";
      }
    }
    head.addEventListener("click", function () { setOpen(!card.classList.contains("is-open")); });
    body.addEventListener("transitionend", function (e) {
      if (e.propertyName !== "max-height") return;
      if (card.classList.contains("is-open")) body.style.maxHeight = "none";
      else body.hidden = true;
    });
    body.addEventListener("click", function (e) {
      var btn = e.target.closest ? e.target.closest("[data-collapse],[data-copy]") : null;
      if (!btn) return;
      if (btn.hasAttribute("data-collapse")) {
        body.style.maxHeight = body.scrollHeight + "px"; /* re-anchor before closing */
        requestAnimationFrame(function () { setOpen(false); });
        head.focus();
      } else if (navigator.clipboard) {
        navigator.clipboard.writeText(btn.getAttribute("data-copy") || "").then(function () {
          var t = btn.textContent; btn.textContent = "Copied";
          setTimeout(function () { btn.textContent = t; }, 1400);
        }).catch(function () {});
      }
    });
  });
  window.addEventListener("resize", function () {
    cards.forEach(function (c) {
      if (c.classList.contains("is-open")) c.querySelector(".card__body").style.maxHeight = "none";
    });
  }, { passive: true });

  /* ---------- filtering (debounced) ---------- */
  var status = "all", term = "";
  function apply() {
    var shown = 0;
    for (var i = 0; i < cards.length; i++) {
      var c = cards[i];
      var okS = status === "all" || c.dataset.status === status;
      var okT = !term || (c.dataset.name || "").toLowerCase().indexOf(term) !== -1;
      var vis = okS && okT;
      c.classList.toggle("is-hidden", !vis);
      if (vis) shown++;
    }
    empty.classList.toggle("is-on", shown === 0);
    count.textContent = shown + " of " + cards.length + " sources";
    if (clear) clear.hidden = !term;
  }
  var t;
  function debounced() { clearTimeout(t); t = setTimeout(function () { term = input.value.trim().toLowerCase(); apply(); }, 120); }
  if (input) {
    input.addEventListener("input", debounced);
    input.addEventListener("keydown", function (e) { if (e.key === "Escape") { input.value = ""; term = ""; apply(); } });
  }
  if (clear) clear.addEventListener("click", function () { input.value = ""; term = ""; apply(); input.focus(); });
  chips.forEach(function (chip) {
    chip.addEventListener("click", function () {
      status = chip.dataset.status;
      chips.forEach(function (c) { c.setAttribute("aria-pressed", String(c === chip)); });
      apply();
    });
  });
  apply();
})();
</script>
</body>
</html>`
