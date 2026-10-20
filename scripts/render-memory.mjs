// Turn recorded Windows process counters into the README's memory figure.
import {readFileSync,writeFileSync} from 'node:fs';
import {createRequire} from 'node:module';
const {Resvg}=createRequire(import.meta.url)(process.env.WISP_RESVG || '@resvg/resvg-js');
const report=JSON.parse(readFileSync(process.argv[2] || 'docs/assets/memory-sample.json','utf8'));
const mib=bytes=>(bytes/1048576).toFixed(1);
const ui=report.summary.wispWorkingSetBytes;
const helper=report.summary.helperWorkingSetBytes;
const total=report.summary.combinedWorkingSetBytes;
const duration=Math.round(report.durationSeconds);
const fraction=ui.mean/(ui.mean+helper.mean);
const helperNames=[...new Set(report.samples.flatMap(sample=>sample.helpers.map(process=>process.name.replace(/\.exe$/,''))))].join(' + ') || 'none observed';
const escape=text=>String(text).replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;').replaceAll('"','&quot;');
const svg=`<svg xmlns="http://www.w3.org/2000/svg" width="1100" height="425" viewBox="0 0 1100 425" role="img" aria-label="Observed memory: wisp ${mib(ui.mean)} MiB, audio helpers ${mib(helper.mean)} MiB; combined peak ${mib(total.max)} MiB">
<defs><linearGradient id="aurora"><stop stop-color="#63F6DF"/><stop offset=".5" stop-color="#78DEFF"/><stop offset="1" stop-color="#CA9BFA"/></linearGradient></defs>
<rect width="1100" height="425" rx="12" fill="#0C0F16"/>
<rect x=".5" y=".5" width="1099" height="424" rx="12" stroke="#293246" fill="none"/>
<path d="M43 47L52 72L61 57L70 72L79 47" stroke="url(#aurora)" stroke-width="4" fill="none"/>
<g font-family="Consolas,DejaVu Sans Mono,monospace">
<text x="99" y="66" font-size="24" fill="#E4EAF1">Small footprint.</text>
<text x="1056" y="65" text-anchor="end" font-size="14" fill="#929CAF">${duration}s observation / Windows ${escape(report.architecture)}</text>
<text x="44" y="110" font-size="15" fill="#929CAF">${escape(report.version)} / observed resident memory</text>
<text x="44" y="172" font-size="16" fill="#87DCEF">wisp.exe</text>
<text x="44" y="227" font-size="48" fill="#E4EAF1">${mib(ui.mean)}<tspan dx="12" font-size="23" fill="#929CAF">MiB</tspan></text>
<text x="568" y="172" font-size="16" fill="#B5A3E8">Audio helpers / ${escape(helperNames)}</text>
<text x="568" y="227" font-size="48" fill="#E4EAF1">${mib(helper.mean)}<tspan dx="12" font-size="23" fill="#929CAF">MiB</tspan></text>
<text x="44" y="256" font-size="13" fill="#929CAF">Mean process working set</text>
<text x="568" y="256" font-size="13" fill="#929CAF">Mean process working set</text>
<rect x="44" y="285" width="${(1012*fraction).toFixed(2)}" height="5" fill="#87DCEF"/>
<rect x="${(44+1012*fraction).toFixed(2)}" y="285" width="${(1012*(1-fraction)).toFixed(2)}" height="5" fill="#B5A3E8"/>
<text x="44" y="331" font-size="18" fill="#E4EAF1">${mib(total.max)} MiB <tspan font-size="15" fill="#929CAF">combined peak in this sample</tspan></text>
<text x="44" y="381" font-size="13" fill="#929CAF">${report.sampleCount} samples / terminal host excluded / memory varies by session</text>
</g></svg>`;
writeFileSync('docs/assets/memory.svg',svg);
writeFileSync('docs/assets/memory.png',new Resvg(svg,{font:{loadSystemFonts:true}}).render().asPng());
console.log(`Memory figure: wisp ${mib(ui.mean)} MiB; helpers ${mib(helper.mean)} MiB; combined peak ${mib(total.max)} MiB.`);
