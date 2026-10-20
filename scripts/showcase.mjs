// Render the real preview views; fixtures never start playback or save a profile.
import {execFileSync} from 'node:child_process';
import {createRequire} from 'node:module';
import {mkdirSync,writeFileSync} from 'node:fs';
import path from 'node:path';
import {terminalSvg} from './preview.mjs';

const {Resvg}=createRequire(import.meta.url)(process.env.WISP_RESVG || '@resvg/resvg-js');
const binary=path.resolve(process.argv[2] || 'dist/wisp.exe');
const out=path.resolve('docs/assets');
mkdirSync(out,{recursive:true});
for(const [name,stage,theme] of [
 ['player','butterfly-search','aurora'],
 ['afterglow','butterfly-search','afterglow'],
 ['lyrics','lyrics','aurora'],
 ['welcome','welcome','afterglow'],
]){
 const raw=execFileSync(binary,['--preview',stage,'--preview-theme',theme],{encoding:'utf8'});
 let svg=terminalSvg(raw);
 // A thin terminal title bar frames the capture without changing its UI content.
 const match=svg.match(/width="(\d+)" height="(\d+)" viewBox="0 0 (\d+) (\d+)"/);
 const width=Number(match[1]),height=Number(match[2]);
 svg=svg.replace(match[0],`width="${width}" height="${height+36}" viewBox="0 0 ${width} ${height+36}"`);
 svg=svg.replace(/(<svg[^>]*>)/,'$1<rect width="100%" height="36" fill="#151a25"/><text x="24" y="24" font-family="Consolas,monospace" font-size="13" fill="#929caf">wisp</text><g transform="translate(0 36)">').replace('</svg>','</g></svg>');
 writeFileSync(path.join(out,`${name}.png`),new Resvg(svg,{font:{loadSystemFonts:true}}).render().asPng());
}

// Match the app's Aurora W silhouette and four-stop diagonal gradient.
const banner=`<svg xmlns="http://www.w3.org/2000/svg" width="1280" height="430" viewBox="0 0 1280 430" role="img" aria-label="wisp — a little room for sound">
<defs><linearGradient id="aurora" x1="0" y1="0" x2="1" y2="1"><stop stop-color="#63F6DF"/><stop offset=".36" stop-color="#78DEFF"/><stop offset=".7" stop-color="#888CF4"/><stop offset="1" stop-color="#CA9BFA"/></linearGradient><linearGradient id="line"><stop stop-color="#63F6DF"/><stop offset=".6" stop-color="#888CF4"/><stop offset="1" stop-color="#CA9BFA"/></linearGradient></defs>
<rect width="1280" height="430" fill="#0C0F16"/>
<path d="M115 121 L175 291 L235 182 L295 291 L355 121" fill="none" stroke="url(#aurora)" stroke-width="27" stroke-linejoin="miter"/>
<g font-family="Consolas,DejaVu Sans Mono,monospace"><text x="440" y="210" fill="#E4EAF1" font-size="112" letter-spacing="-6">wisp</text><text x="447" y="259" fill="#A4AEC3" font-size="24">a little room for sound</text>
<text x="447" y="334" fill="#7C8AA4" font-size="15">Music, at home in your terminal.</text></g>
<path d="M110 378 H1166" stroke="#293246"/>
<path d="M110 378 H520" stroke="url(#line)" stroke-width="2"/>
<g fill="none" stroke="url(#aurora)" stroke-width="2" transform="translate(1050 157)"><path d="M0 21 C-35-20-54 7-17 31 C-47 50-18 63 0 29 C18 63 47 50 17 31 C54 7 35-20 0 21 Z"/><path d="M0 18V44 M0 18L-6 10 M0 18L6 10"/></g></svg>`;
writeFileSync(path.join(out,'banner.svg'),banner);
writeFileSync(path.join(out,'banner.png'),new Resvg(banner,{font:{loadSystemFonts:true}}).render().asPng());
console.log('Rendered banner and four application previews in docs/assets.');
