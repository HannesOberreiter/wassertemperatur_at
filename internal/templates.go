package internal

const pageTemplates = `
{{define "head"}}
<!doctype html>
<html lang="de">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>{{.Title}}</title>
	<meta name="description" content="{{.Description}}">
	<link rel="icon" type="image/png" href="/assets/icons/favicon.png">
	<link rel="apple-touch-icon" href="/assets/icons/favicon.png">
	<link rel="stylesheet" href="/assets/css/main.css">
	<script src="/assets/js/htmx.min.js"></script>
</head>
<body>
<header class="site-header">
	<a class="brand" href="/">Wassertemperatur Österreich</a>
	<nav>
		<a href="/">Temperatur</a>
		<a href="/wasserqualitaet">Wasserqualität</a>
		<a href="/impressum">Impressum</a>
	</nav>
</header>
{{end}}

{{define "foot"}}
<section class="sponsor">
	<a href="https://www.aquavital.at/" target="_blank" rel="external" referrerpolicy="origin">
		<span>Domain Sponsor:</span>
		<img src="/assets/img/logo_aquavital.png" alt="Sponsor Aquavital Kalkmagnet" width="150" height="53">
	</a>
</section>

</body>
</html>
{{end}}

{{define "index"}}
{{template "head" .}}
<main class="container">
	<section>
		<h1>Aktuelle Wassertemperaturen in Österreich</h1>
		<p>Übersicht über aktuelle Temperaturen von Seen, Flüssen und Badegewässern. Die Wochenänderung vergleicht den Durchschnitt der letzten sieben Tage mit den sieben Tagen davor.</p>
	</section>
	{{template "filters" .}}
	{{template "table" .}}
	<section>
		<h2>Wassertemperaturen vergleichen</h2>
		<p>Wassertemperaturen hängen von Wetter, Zufluss, Wind und Tiefe ab. Große Seen reagieren langsamer, kleine Badeplätze schneller. Der Verlauf auf den Detailseiten hilft beim Einordnen.</p>
	</section>
</main>
{{template "foot" .}}
{{end}}

{{define "quality"}}
{{template "head" .}}
<main class="container">
	<section>
		<h1>Wasserqualität österreichischer Badegewässer</h1>
		<p>Diese Seite zeigt Badegewässer mit Qualitätsbewertung und Sichttiefe. Temperaturwerte bleiben auf der separaten Temperaturseite.</p>
	</section>
	{{template "filters" .}}
	{{template "qualityTable" .}}
</main>
{{template "foot" .}}
{{end}}

{{define "filters"}}
<form class="filters" hx-get="{{if .QualityPage}}/qualitaet-tabelle{{else}}/table{{end}}" hx-target="#table" hx-push-url="true">
	<label>Suche <input name="suche" value="{{.Search}}" placeholder="z. B. Attersee" hx-trigger="keyup changed delay:400ms" hx-get="{{if .QualityPage}}/qualitaet-tabelle{{else}}/table{{end}}" hx-target="#table" hx-push-url="true" hx-include="closest form"></label>
	<label>Bundesland
		<select name="bundesland" hx-trigger="change" hx-get="{{if .QualityPage}}/qualitaet-tabelle{{else}}/table{{end}}" hx-target="#table" hx-push-url="true" hx-include="closest form">
			<option value="">Alle</option>
			{{range .States}}<option value="{{.}}" {{if eq $.State .}}selected{{end}}>{{.}}</option>{{end}}
		</select>
	</label>
	<button type="submit">Filtern</button>
</form>
{{end}}

{{define "table"}}
<div id="table">
	<p><strong>{{len .Waters}}</strong> Gewässer gefunden</p>
	<table>
		<thead><tr><th>Gewässer</th><th>Bundesland</th><th>Datum</th><th>Temperatur</th><th>Wochenänderung</th></tr></thead>
		<tbody>
		{{range .Waters}}
		<tr class="{{if not .Recent}}old{{end}}">
			<td><a href="{{detailURL .}}">{{.Name}}</a><small>{{.Source}}</small></td>
			<td>{{.State}}</td>
			<td>{{date .MeasuredAt}}</td>
			<td class="num">{{temp .Temperature}}</td>
			<td class="num">{{change .Change}}</td>
		</tr>
		{{else}}
		<tr><td colspan="5">Keine Gewässer gefunden.</td></tr>
		{{end}}
		</tbody>
	</table>
</div>
{{end}}

{{define "qualityTable"}}
<div id="table">
	<p><strong>{{len .Waters}}</strong> Badegewässer gefunden</p>
	<table>
		<thead><tr><th>Gewässer</th><th>Bundesland</th><th>Datum</th><th>Sichttiefe</th><th>Qualität</th></tr></thead>
		<tbody>
		{{range .Waters}}
		<tr class="{{if not .Recent}}old{{end}}">
			<td><a href="{{detailURL .}}">{{.Name}}</a></td>
			<td>{{.State}}</td>
			<td>{{date .MeasuredAt}}</td>
			<td class="num">{{depth .Depth}}</td>
			<td class="num">{{quality .Quality}}</td>
		</tr>
		{{else}}
		<tr><td colspan="5">Keine Badegewässer gefunden.</td></tr>
		{{end}}
		</tbody>
	</table>
	<p class="hint">Qualität: 1 Ausgezeichnete Badegewässerqualität, 2 Gute Badegewässerqualität, 3 Mangelhafte Badegewässerqualität, 4 Baden verboten / vom Baden wird abgeraten.</p>
	<p class="hint">Sichttiefe zeigt, wie tief man ins Wasser sehen kann. Enterokokken und E. coli sind Bakterienwerte und Hinweise auf mögliche fäkale Verunreinigung; niedrige Werte sind besser.</p>
</div>
{{end}}

{{define "detail"}}
{{template "head" .}}
<main class="container">
	<p><a href="/">Zurück zur Übersicht</a></p>
	<h1>Wassertemperatur {{.Water.Name}}</h1>
	<p>{{.Water.Name}} liegt in {{.Water.State}}. Die Tabelle zeigt die gespeicherten Messungen.</p>
	{{if .QualityDetail}}
	<table>
		<thead><tr><th>Datum</th><th>Temperatur</th><th>Sichttiefe</th><th>Qualität</th><th>Enterokokken</th><th>E. coli</th></tr></thead>
		<tbody>
		{{range .Observations}}
		<tr><td>{{day .MeasuredAt}}</td><td class="num">{{temp .Temperature}}</td><td class="num">{{depth .Depth}}</td><td class="num">{{quality .Quality}}</td><td class="num">{{param .EnterococciOp .Enterococci}}</td><td class="num">{{param .EColiOp .EColi}}</td></tr>
		{{else}}
		<tr><td colspan="6">Noch kein Verlauf gespeichert.</td></tr>
		{{end}}
		</tbody>
	</table>
	<p class="hint">Qualität: 1 Ausgezeichnete Badegewässerqualität, 2 Gute Badegewässerqualität, 3 Mangelhafte Badegewässerqualität, 4 Baden verboten / vom Baden wird abgeraten.</p>
	<p class="hint">Sichttiefe zeigt, wie tief man ins Wasser sehen kann. Enterokokken und E. coli sind Bakterienwerte und Hinweise auf mögliche fäkale Verunreinigung; niedrige Werte sind besser.</p>
	{{else}}
	<table>
		<thead><tr><th>Tag</th><th class="num">Durchschnitt</th><th class="num">Median</th><th class="num">Hoch</th><th class="num">Tief</th><th class="num">Messungen</th></tr></thead>
		<tbody>
		{{range .Summaries}}
		<tr><td>{{day .Day}}</td><td class="num">{{plainTemp .Avg}}</td><td class="num">{{plainTemp .Median}}</td><td class="num">{{plainTemp .High}}</td><td class="num">{{plainTemp .Low}}</td><td class="num">{{.Count}}</td></tr>
		{{else}}
		<tr><td colspan="6">Noch kein Verlauf gespeichert.</td></tr>
		{{end}}
		</tbody>
	</table>
	<nav class="pager">
		{{if gt .Page 1}}<a href="?seite={{.PrevPage}}">Neuere Werte</a>{{end}}
		{{if eq (len .Summaries) 30}}<a href="?seite={{.NextPage}}">Ältere Werte</a>{{end}}
	</nav>
	{{end}}
</main>
{{template "foot" .}}
{{end}}

{{define "imprint"}}
{{template "head" .}}
<main class="container narrow">
	<h1>Impressum</h1>
	<p>Informationsangebot von Hannes Oberreiter.</p>
	<h2>Kontakt</h2>
	<p>Hannes Oberreiter<br><a href="https://www.oberreiter.or.at/hannes" target="_blank" rel="external" referrerpolicy="origin">oberreiter.or.at/hannes</a></p>
	<h2>Haftung</h2>
	<p>Die Inhalte werden sorgfältig erstellt, trotzdem sind Fehler oder veraltete Messwerte möglich. Angaben ohne Gewähr.</p>
	<h2>Datenquellen</h2>
	<p>
		<a href="https://ages.at/" target="_blank" rel="external" referrerpolicy="origin">AGES</a><br>
		<a href="https://data.ooe.gv.at/" target="_blank" rel="external" referrerpolicy="origin">Land Oberösterreich</a><br>
		<a href="https://www.steiermark.com/de/Ausseerland-Salzkammergut" target="_blank" rel="external" referrerpolicy="origin">Ausseerland-Salzkammergut</a><br>
		<a href="https://hydrographie.ktn.gv.at/" target="_blank" rel="external" referrerpolicy="origin">Land Kärnten</a><br>
		<a href="https://vowis.vorarlberg.at/stationswrapper" target="_blank" rel="external" referrerpolicy="origin">Wasser Online Vorarlberg</a><br>
		<a href="https://wasser.bgld.gv.at/hydrographie/die-seen" target="_blank" rel="external" referrerpolicy="origin">Wasserportal Burgenland</a>
	</p>
	<h2>Projekt</h2>
	<p>Erstellt von <a href="https://www.oberreiter.or.at/hannes" target="_blank" rel="external" referrerpolicy="origin">Hannes Oberreiter</a>. Code auf <a href="https://github.com/HannesOberreiter/wassertemperatur_at" target="_blank" rel="external" referrerpolicy="origin">GitHub</a>.</p>
</main>
{{template "foot" .}}
{{end}}
`
