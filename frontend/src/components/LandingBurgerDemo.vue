<script setup>
import AppIcon from './AppIcon.vue';
defineProps({ phase: Number, playing: Boolean });
</script>
<template>
 <div class="burger-demo" :data-phase="phase" :data-playing="playing" aria-label="Ilustración del catálogo de LeguiBurger, con datos de ejemplo">
  <div class="demo-orbit" aria-hidden="true" />
  <div class="demo-ticket"><div class="ticket-brand"><AppIcon name="burger" :size="28" /><span>leguiburger.<small>EL SABOR DE HACERLO BIEN</small></span></div><div class="ticket-divider" /><p class="ticket-eyebrow">LA RECETA DE UN BUEN DÍA</p><h2>Todo a punto.</h2><div class="ticket-line"><span>Tu carta</span><strong>Irresistible</strong></div><div class="ticket-line"><span>Tu equipo</span><strong>Organizado</strong></div><div class="ticket-line"><span>Tu cocina</span><strong>En orden</strong></div><div class="ticket-divider" /><div class="ticket-total"><span>El ingrediente secreto</span><strong>Tu negocio,<br />bajo control.</strong></div><div class="ticket-barcode" /><small class="ticket-bottom">HECHO CON GANAS · SERVIDO CON ORDEN</small></div>
  <div class="demo-menu-card"><div class="demo-card-header"><span class="demo-icon"><AppIcon name="burger" /></span><div><strong>La carta de la casa</strong><small>Catálogo de tu marca</small></div><span class="demo-badge">Vista de ejemplo</span></div><div class="demo-food">
   <svg viewBox="0 0 360 260" role="img" aria-label="Hamburguesa doble con queso" class="burger-visual"><defs><linearGradient id="legui-bun" x2="0" y2="1"><stop stop-color="#efb454" /><stop offset="1" stop-color="#b75e22" /></linearGradient><linearGradient id="legui-meat" x2="0" y2="1"><stop stop-color="#643420" /><stop offset="1" stop-color="#301c17" /></linearGradient><radialGradient id="legui-glow"><stop stop-color="#ce8737" stop-opacity=".35" /><stop offset="1" stop-color="#ce8737" stop-opacity="0" /></radialGradient></defs><ellipse cx="180" cy="142" rx="165" ry="125" fill="url(#legui-glow)" /><ellipse cx="181" cy="232" rx="115" ry="11" fill="#150e09" opacity=".3" /><g class="burger-layers"><path d="M67 204q113-23 226 0l-7 19q-5 13-106 13T74 223z" fill="url(#legui-bun)" /><path d="M62 191q20-17 43-9t42-1 48 4 48-5 54 13l-5 14q-99 15-224 0z" fill="url(#legui-meat)" /><path d="M62 177l230-2-38 27-48-13-39 14-55-19-39 12z" fill="#f7b632" /><path d="M65 166q113-24 228-1l-4 17q-23 10-46 3t-46 1-46-3-46 4-40-5z" fill="url(#legui-meat)" /><path d="M65 153l229 1-31 24-39-8-40 20-37-19-39 10-43-18z" fill="#f5c247" /><path d="M67 145q12-9 28 0t28-1 28 2 28-3 28 4 28-3 28 2 30 2" stroke="#b5442b" stroke-width="14" fill="none" /><path d="M65 132q10-14 24-2t28-1 28 2 28-5 28 4 28-3 28 3 37 1l-9 15q-14-10-29 0t-29-3-29 4-29-4-29 1-29-1-29 1z" fill="#718140" /><path d="M66 119c4-58 48-86 114-86s111 28 115 86q-10 14-115 14T66 119z" fill="url(#legui-bun)" /><path d="M78 100c21-44 58-55 100-55" stroke="#f6ce7c" stroke-opacity=".3" stroke-width="4" fill="none" stroke-linecap="round" /><g stroke="#fae0a5" stroke-width="3" stroke-linecap="round"><path d="M125 69l5-3 M165 57l4 2 M205 63l5-2 M238 78l4 3 M96 96l4-2 M151 94l4 2 M186 80l4-2 M216 105l4-2 M264 104l4 2 M173 114l4-1" /></g></g></svg>
   <span class="food-tag">LA FAVORITA</span></div><div class="demo-product-info"><div><h3>Doble clásica</h3><p>Doble carne. Doble felicidad.</p></div><strong>$ 9.800</strong></div><div class="demo-product-foot"><span><i class="live-dot" />Disponible en tu carta</span><AppIcon name="check" :size="17" /></div></div>
  <div class="demo-toast"><span><AppIcon name="check" /></span><div><strong>Todo en su lugar.</strong><small>De la cocina a cada sucursal.</small></div></div>
 </div>
</template>
<style scoped>
.burger-demo { position:relative; min-width:0; height:550px; isolation:isolate; }
.demo-orbit { position:absolute; width:450px; height:450px; max-width:100%; border:1px solid color-mix(in srgb,var(--green) 12%,transparent); border-radius:50%; top:35px; right:-35px; z-index:-1; animation:orbit-breathe 7s ease-in-out infinite; }
.demo-orbit::after { content:''; position:absolute; inset:35px; border:1px solid color-mix(in srgb,var(--green) 7%,transparent); border-radius:50%; }
.demo-ticket { position:absolute; top:48px; right:-12px; width:270px; background:#f5e7ce; color:#3b2b1b; padding:27px 23px 25px; transform:rotate(7deg); box-shadow:0 20px 70px #0003; border-radius:3px; animation:ticket-float 7s ease-in-out infinite; }
.ticket-brand { display:flex; gap:8px; align-items:center; }
.ticket-brand > span { font-weight:750; font-size:20px; letter-spacing:-.8px; }
.ticket-brand small { display:block; font-size:6px; letter-spacing:1.1px; font-weight:500; }
.ticket-divider { border-top:1px dashed #b4a083; margin:22px 0; }
.ticket-eyebrow { color:#8a7354; font-size:8px; margin:0 0 7px; letter-spacing:.1em; }
.demo-ticket h2 { font-size:24px; margin:0 0 12px; }
.ticket-line { display:flex; justify-content:space-between; font-size:10px; padding:9px 0; transition:opacity .55s,transform .55s; }
.ticket-line strong { font-weight:500; }
[data-phase="0"] .ticket-line { opacity:.35; transform:translateY(4px); }
.ticket-line:nth-of-type(4) { transition-delay:.1s; }
.ticket-line:nth-of-type(5) { transition-delay:.2s; }
.ticket-total > span { display:block; text-transform:uppercase; font-size:7px; letter-spacing:.08em; margin-bottom:10px; }
.ticket-total strong { font-size:25px; line-height:1.1; display:block; letter-spacing:-1px; }
.ticket-barcode { height:24px; margin:28px 20px 10px; background:repeating-linear-gradient(90deg,#443522 0px,#443522 2px,transparent 2px,transparent 5px,#443522 5px,#443522 6px,transparent 6px,transparent 9px); opacity:.75; }
.ticket-bottom { display:block; text-align:center; font-size:6px; letter-spacing:1px; }
.demo-menu-card { position:absolute; top:92px; left:-8px; width:310px; background:var(--surface); border:1px solid var(--border); border-radius:16px; box-shadow:0 25px 70px #0004; transform:rotate(-5deg); animation:menu-float 6s ease-in-out infinite; }
.demo-card-header { display:flex; align-items:center; gap:9px; padding:16px; }
.demo-icon { display:grid; place-items:center; width:31px; height:31px; background:var(--hover); color:var(--green); border-radius:8px; }
.demo-card-header strong { display:block; font-size:11px; font-weight:550; }
.demo-card-header small { font-size:8px; color:var(--muted); }
.demo-badge { font-size:6px; color:var(--muted); margin-left:auto; white-space:nowrap; }
.demo-food { position:relative; background:#33271c; overflow:hidden; }
.burger-visual { width:100%; height:230px; display:block; }
.burger-layers { animation:burger-breathe 4s ease-in-out infinite; transform-origin:center; }
.food-tag { position:absolute; top:14px; left:16px; border:1px solid #a4743b; background:#312419; border-radius:4px; padding:4px 7px; font-size:7px; letter-spacing:.1em; color:#f3c079; }
.demo-product-info { display:flex; align-items:center; justify-content:space-between; padding:18px 16px; gap:8px; }
.demo-product-info h3 { font-size:17px; margin:0 0 6px; }
.demo-product-info p { margin:0; font-size:9px; color:var(--muted); }
.demo-product-info > strong { font-size:14px; color:var(--green); white-space:nowrap; }
.demo-product-foot { padding:11px 16px; border-top:1px solid var(--border); display:flex; justify-content:space-between; align-items:center; font-size:8px; color:var(--muted); }
.demo-product-foot span { display:flex; align-items:center; gap:6px; }
.demo-product-foot > svg { color:var(--green); }
.demo-toast { position:absolute; bottom:13px; right:7px; background:var(--surface); border:1px solid var(--border); box-shadow:0 15px 40px #0003; padding:14px 20px; border-radius:9px; display:flex; align-items:center; gap:12px; transition:opacity .5s,transform .6s cubic-bezier(.16,1,.3,1); }
[data-phase="0"] .demo-toast,[data-phase="1"] .demo-toast { opacity:0; transform:translateY(14px) scale(.96); }
.demo-toast > span { display:grid; place-items:center; width:30px; height:30px; border-radius:50%; background:var(--hover); color:var(--green); }
.demo-toast strong { display:block; font-size:12px; }
.demo-toast small { font-size:9px; color:var(--muted); }
[data-playing=false] * { animation-play-state:paused !important; }
@keyframes menu-float { 0%,100% { transform:rotate(-5deg) translateY(0); } 50% { transform:rotate(-3.5deg) translateY(-12px); } }
@keyframes ticket-float { 0%,100% { transform:rotate(7deg) translateY(0); } 50% { transform:rotate(5deg) translateY(8px); } }
@keyframes orbit-breathe { 0%,100% { opacity:.5; transform:scale(.96); } 50% { opacity:1; transform:scale(1.04); } }
@keyframes burger-breathe { 0%,100% { transform:translateY(0); } 50% { transform:translateY(-4px); } }
@media(max-width:1100px) and (min-width:851px) { .burger-demo { transform:scale(.85); transform-origin:left center; width:116%; } }
@media(max-width:850px) { .burger-demo { width:min(100%,500px); margin:auto; } }
@media(max-width:540px) { .burger-demo { height:475px; } .demo-menu-card { width:250px; top:80px; left:-3px; } .burger-visual { height:180px; } .demo-ticket { width:230px; right:-5px; top:35px; padding:20px 17px; } .ticket-brand > span { font-size:17px; } .demo-ticket h2 { font-size:21px; } .ticket-total strong { font-size:20px; } .ticket-divider { margin-block:17px; } .demo-toast { right:-3px; bottom:0; padding:12px 15px; } .demo-card-header { padding:13px; } .demo-badge { display:none; } .demo-orbit { right:-10px; top:40px; width:330px; height:330px; } }
@media(max-width:360px) { .burger-demo { height:420px; } .demo-menu-card { width:220px; top:70px; } .demo-ticket { width:200px; font-size:9px; top:20px; } .burger-visual { height:155px; } .demo-product-info { padding:13px; } .demo-product-info h3 { font-size:15px; } .demo-ticket h2 { font-size:19px; } .ticket-barcode { margin-top:18px; height:17px; } .ticket-divider { margin-block:12px; } .ticket-total strong { font-size:18px; } .ticket-brand > span { font-size:14px; } .ticket-brand small,.ticket-bottom { font-size:5px; } }
</style>
