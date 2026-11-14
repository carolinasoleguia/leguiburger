<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import LandingBurgerDemo from '../../components/LandingBurgerDemo.vue';
import BrandLogo from '../../components/BrandLogo.vue';
import ThemeToggle from '../../components/ThemeToggle.vue';
import AppIcon from '../../components/AppIcon.vue';
import { useTheme } from '../../composables/useTheme.js';
import { mountLandingWheel } from '../../composables/landingWheel.js';

const { theme } = useTheme();
const root = ref(null), menuButton = ref(null), menuOpen = ref(false), playing = true, phase = ref(0), active = ref(0), panelView = ref('Catálogo'), openFaq = ref(null), teamView = ref(false);
const nav = [{ href:'#como-funciona', label:'Cómo funciona' }, { href:'#tu-espacio', label:'Tu espacio' }, { href:'#equipo', label:'Tu equipo' }, { href:'#preguntas', label:'Preguntas' }];
const waveform = Array.from({ length:27 }, (_, i) => ({ height: `${[8,16,25,13,32,21,11,28,18][i%9]}px`, animationDelay:`${i*-.09}s` }));
const example = { business:'LeguiBurger · Local Centro', job:'Producción de la jornada', message:'Para el servicio de hoy: 120 medallones, 120 panes y 30 porciones de papas. Todo preparado para el turno de la noche.', items:[{ name:'Medallones preparados', amount:'120 u.' },{ name:'Panes comprados', amount:'120 u.' },{ name:'Porciones de papas', amount:'30 u.' }] };
const storySteps = [{ title:'Organizá tu negocio', text:'Tus marcas, tus locales y las personas que los administran.' },{ title:'Prepará tu carta', text:'Cargá los productos y ajustá precios y disponibilidad por sucursal.' },{ title:'Poné la cocina en orden', text:'Registrá la producción y organizá los accesos de tu equipo.' }];
const panelViews = ['Catálogo','Locales','Producción'];
const panels = {
 Catálogo:{ copy:'Consultá los productos de tu marca y ajustá la oferta, los precios y la disponibilidad para cada local.', headings:['Producto','Estado','Precio'], rows:[['Doble clásica','Disponible','$ 9.800'],['Cheddar & bacon','Disponible','$ 11.500'],['Papas de la casa','Disponible','$ 4.200']], subtitles:['La favorita de la casa','Con bacon crocante','Para acompañar'] },
 Locales:{ copy:'Encontrá las sucursales de tu marca y administrá la oferta de cada una desde el mismo lugar.', headings:['Local','Marca','Catálogo'], rows:[['Local Centro','LeguiBurger','Personalizado'],['Local Norte','LeguiBurger','Personalizado'],['Local Sur','LeguiBurger','Personalizado']], subtitles:['Sucursal Centro','Sucursal Norte','Sucursal Sur'] },
 Producción:{ copy:'Consultá los reportes diarios de tu cocina, las cantidades preparadas y las notas de cada jornada.', headings:['Jornada','Medallones','Panes'], rows:[['Lunes','120 u.','120 u.'],['Martes','160 u.','160 u.'],['Miércoles','140 u.','150 u.']], subtitles:['Producción del día','Producción del día','Producción del día'] },
};
const faqs = [
 { question:'¿Qué puedo gestionar con LeguiBurger?', answer:'Tus marcas, locales, administradores, equipo, catálogo y reportes de producción. Cada persona tiene un espacio de trabajo según su rol.' },
 { question:'¿Puedo trabajar con más de un local?', answer:'Sí. Podés organizar varios locales y ajustar los precios y la disponibilidad de los productos para cada sucursal.' },
 { question:'¿Mi equipo puede ingresar al sistema?', answer:'Sí. LeguiBurger tiene accesos de propietario, administrador y empleado para organizar el trabajo de cada persona.' },
 { question:'¿Necesito instalar algo?', answer:'Entrás desde el navegador de tu computadora o celular con tu cuenta de LeguiBurger, sin instalar otra aplicación.' },
];
const roleCards = computed(() => [
 { name:'Tu marca', description:'Una mirada completa de tu negocio.', label:'Dueño', period:'Marcas, sucursales y personas.', features:['Gestión de marcas','Administración de locales','Accesos para administradores','Resumen del negocio'], cta:'Entrar a mi negocio' },
 { name:'Tus locales', description:'El día a día, en un mismo lugar.', label:teamView.value ? 'En equipo' : 'A tu ritmo', period:'El espacio de cada administrador.', features:['Catálogo general de la marca','Precios y disponibilidad por sucursal','Gestión de empleados','Reportes de producción'], cta:'Abrir mi espacio' },
 { name:'Tu cocina', description:'Personas y producción, organizadas.', label:'Equipo', period:'Cada persona con su acceso.', features:['Acceso de empleados','Roles de trabajo definidos','Un espacio para cada persona','Todo desde el navegador'], cta:'Ingresar al sistema' },
]);
let dispose = () => {};
onMounted(() => {
 const element = root.value;
 const observer = new IntersectionObserver(entries => entries.forEach(entry => {
  if (!entry.isIntersecting) return;
  entry.target.dataset.entry='done'; observer.unobserve(entry.target);
 }), { threshold:.12, rootMargin:'0px 0px -6% 0px' });
 element.querySelectorAll('[data-reveal], [data-text-motion]:not([data-intro=true])').forEach(node => { node.dataset.entry='pending'; observer.observe(node); });
 const visible = new Set();
 const playback = new IntersectionObserver(entries => entries.forEach(entry => {
  const key = entry.target.dataset.playback;
  if (entry.isIntersecting && entry.intersectionRatio >= .3) visible.add(key); else visible.delete(key);
  entry.target.dataset.inView = visible.has(key);
 }), { threshold:[.3] });
 element.querySelectorAll('[data-playback]').forEach(node => playback.observe(node));
 let demoTimer, storyTimer, panelTimer;
 function sync() {
  clearInterval(demoTimer); clearInterval(storyTimer); clearInterval(panelTimer);
  if (document.hidden) return;
  demoTimer=setInterval(()=>{ if(playing && visible.has('demo')) phase.value=(phase.value+1)%4; },2600);
  storyTimer=setInterval(()=>{ if(playing && visible.has('story') && !element.querySelector('.storySteps:focus-within, .storySteps:hover')) active.value=(active.value+1)%3; },2600);
  panelTimer=setInterval(()=>{ if(playing && visible.has('panel') && !element.querySelector('.panelControls:focus-within, .panelControls:hover')) panelView.value=panelViews[(panelViews.indexOf(panelView.value)+1)%3]; },8000);
 }
 sync(); document.addEventListener('visibilitychange',sync);
 const stopWheel=mountLandingWheel(element);
 const escape=event=>{ if(event.key==='Escape' && menuOpen.value){ menuOpen.value=false; menuButton.value?.focus(); } };
 document.addEventListener('keydown',escape);
 dispose=()=>{ observer.disconnect(); playback.disconnect(); clearInterval(demoTimer); clearInterval(storyTimer); clearInterval(panelTimer); stopWheel?.(); document.removeEventListener('visibilitychange',sync); document.removeEventListener('keydown',escape); };
});
onBeforeUnmount(()=>dispose());
</script>

<template>
 <main ref="root" class="landing" :data-theme="theme" data-landing-motion data-motion="full" :data-paused="!playing">
  <a class="skipLink" href="#contenido">Ir al contenido</a>
  <header class="header"><div class="headerInner"><BrandLogo /><nav class="desktopNav" aria-label="Navegación principal"><a v-for="item in nav" :key="item.href" :href="item.href">{{ item.label }}</a></nav><div class="headerActions"><ThemeToggle /><router-link class="login" to="/login">Ingresar<AppIcon name="arrow" :size="17" /></router-link></div><button ref="menuButton" class="menuButton" type="button" :aria-label="menuOpen ? 'Cerrar menú' : 'Abrir menú'" :aria-expanded="menuOpen" aria-controls="landing-menu" @click="menuOpen=!menuOpen"><AppIcon :name="menuOpen ? 'close' : 'menu'" /></button></div><nav v-if="menuOpen" id="landing-menu" class="mobileNav" aria-label="Navegación móvil"><a v-for="item in nav" :key="item.href" :href="item.href" @click="menuOpen=false">{{ item.label }}</a><router-link to="/login">Ingresar a mi cuenta</router-link></nav></header>

  <section id="contenido" class="hero" aria-labelledby="hero-title"><div class="heroCopy"><div class="heroEyebrow"><span />Para tu hamburguesería</div><h1 id="hero-title"><span class="textMotion" data-text-motion data-intro="true">Tu negocio bajo control,</span><br /><em>con todo el sabor de tu marca.</em></h1><p>Tu marca, tus locales y tu cocina, en un mismo lugar. Organizá el catálogo, los precios por sucursal, el equipo y la producción diaria. Todo listo para acompañar el ritmo de tu hamburguesería.</p><div class="heroActions"><router-link to="/login" class="primary">Entrar a mi negocio<AppIcon name="arrow" :size="18" /></router-link><a href="#como-funciona" class="textLink">Ver cómo funciona<span aria-hidden="true">↓</span></a></div><p class="heroNote"><AppIcon name="check" :size="15" />Todo desde tu navegador<span>·</span>Con tu cuenta de LeguiBurger</p></div>
   <div class="demoStage" data-playback="demo" :data-phase="phase" :data-playing="playing"><LandingBurgerDemo :phase="phase" :playing="playing" />
    <div class="demoControls"><div class="demoProgress" aria-hidden="true"><span v-for="(label,index) in ['Tu carta','Tu negocio','Todo a punto']" :key="label" :data-active="(phase===0 ? 0 : phase===1 ? 1 : 2) === index"><i />{{ label }}</span></div></div><p class="illustrationNote">Diseño ilustrativo con datos de ejemplo.</p>
   </div>
  </section>

  <section id="como-funciona" class="scrollStory" :data-story-step="active" data-playback="story" aria-labelledby="process-title"><div class="storySticky"><div class="storyCopy"><h2 id="process-title" data-reveal>Vos ponés el sabor.<br /><span>El negocio, en orden.</span></h2><p class="storyIntro" data-reveal>Desde tu marca hasta lo que pasa en la cocina, seguí cada parte del negocio en un solo lugar.</p><ol class="storySteps"><li v-for="(step,index) in storySteps" :key="step.title" :data-active="active===index"><button type="button" :aria-current="active===index ? 'step' : undefined" @click="active=index"><span class="stepNumber">{{ index+1 }}</span><span><strong>{{ step.title }}</strong><span>{{ step.text }}</span></span></button></li></ol></div>
   <div class="storyVisual" data-ui="workflow-example" aria-hidden="true"><div class="storyOrbit" /><div class="storyOrbitInner" /><div class="storyAudio"><div class="storyMiniHeader"><AppIcon class="smallIcon" name="store" /><span>Tu negocio</span><span>Hoy</span></div><div class="storyWave"><AppIcon name="flame" :size="22" /><span class="waveform"><i v-for="(style,index) in waveform" :key="index" :style="style" /></span></div><p>“La carta de la casa, el equipo y todo lo que pasa en la cocina.”</p><span class="storyAudioLabel">Desde tu espacio de trabajo</span></div><div class="storyDocument"><div class="storyMiniHeader"><span class="documentLogo"><AppIcon name="burger" :size="15" /></span><strong>LeguiBurger</strong><AppIcon name="file" :size="18" /></div><h3>Tu carta</h3><p>La oferta de tu marca</p><dl><div><dt>Doble clásica</dt><dd>$ 9.800</dd></div><div><dt>Cheddar & bacon</dt><dd>$ 11.500</dd></div><div><dt>Papas de la casa</dt><dd>$ 4.200</dd></div></dl><div class="total"><span>Catálogo</span><strong>A tu medida.</strong></div><div class="storyDocumentFooter"><AppIcon name="check" :size="14" />Tu marca. Tu sabor.</div></div><div class="storySent"><span><AppIcon name="file" :size="20" />Catálogo de la marca</span><span>Precios por sucursal<AppIcon name="check" :size="15" /></span></div><div class="storyPaid"><span class="paidCheck"><AppIcon name="check" :size="30" /></span><p>Jornada organizada</p><strong>Todo a punto.</strong><span>Tu equipo y tu cocina, en un mismo lugar.</span><div><span>Tu espacio de trabajo</span><b>En orden<AppIcon name="check" :size="13" /></b></div></div><div class="storyCaption"><span>0{{ active+1 }} / 03</span><span>{{ ['Tu negocio. Así empieza todo.','La carta que lleva tu marca.','Una cocina con todo a punto.'][active] }}</span></div><span class="storySample">Recorrido ilustrativo con datos de ejemplo.</span></div>
  </div></section>

  <section id="tu-espacio" class="panelSection" aria-labelledby="panel-title"><div class="panelIntro"><h2 id="panel-title" data-reveal>La cocina sigue.<br />El orden, también.</h2><p data-reveal>Catálogo, locales, equipo y producción, en un mismo lugar. Consultalo desde el celular o la computadora.</p></div><div class="panelLayout" data-scroll-stop="panel-layout" data-playback="panel"><div class="panelControls" data-reveal><p class="panelControlsLabel">Explorá tu espacio</p><fieldset><legend class="srOnly">Explorá tu espacio de trabajo</legend><label v-for="view in panelViews" :key="view"><input v-model="panelView" type="radio" name="vista-panel" :value="view" /><span><span class="panelOptionTitle">{{ view }}<AppIcon name="arrow" :size="22" /></span><span v-if="panelView===view" class="panelOptionDescription">{{ panels[view].copy }}</span></span></label></fieldset></div><div data-reveal><div :key="panelView" class="panelPreview" :data-view="panelView" data-ui="panel-example" aria-live="polite"><div class="previewHeader"><strong>{{ panelView }}</strong><span>Vista de ejemplo</span></div><table><caption class="srOnly">Ejemplo ilustrativo de {{ panelView }}</caption><thead><tr><th v-for="heading in panels[panelView].headings" :key="heading" scope="col">{{ heading }}</th></tr></thead><tbody><tr v-for="(row,rowIndex) in panels[panelView].rows" :key="row[0]"><td>{{ row[0] }}<span>{{ panels[panelView].subtitles[rowIndex] }}</span></td><td><span :class="panelView==='Catálogo' ? 'statusAccepted' : undefined">{{ row[1] }}</span></td><td>{{ row[2] }}</td></tr></tbody></table><div class="previewFooter"><span>Todo queda en el mismo lugar.</span><AppIcon name="arrow" :size="18" /></div></div></div></div>
   <section class="integrationSection" data-scroll-stop="integrations" aria-labelledby="integration-title"><div><h2 id="integration-title"><span class="textMotion" data-text-motion>Con las personas<br />que hacen tu negocio.</span></h2><p><span class="textMotion" data-text-motion data-variant="body">Una marca, sus locales y el equipo de cada día. Conectá la gestión de tu hamburguesería desde un mismo espacio.</span></p></div><div class="integrationNames" data-reveal><span><span class="integrationMark"><AppIcon name="burger" :size="39" /></span>Tu marca<small>Dale identidad</small></span><span><span class="integrationMark"><AppIcon name="store" :size="39" /></span>Tus locales<small>Organizá cada sucursal</small></span><span><span class="integrationMark"><AppIcon name="flame" :size="39" /></span>Tu cocina<small>Prepará la jornada</small></span></div></section>
  </section>

  <section id="equipo" class="pricing" aria-labelledby="roles-title"><div class="sectionHeading"><h2 id="roles-title"><span class="textMotion" data-text-motion>Un espacio para cada rol.</span></h2><p data-reveal>Cada persona con las herramientas que necesita para acompañar el ritmo de tu negocio.</p></div><fieldset class="billing"><legend>La gestión de tu negocio</legend><label v-for="value in [false,true]" :key="String(value)"><input v-model="teamView" type="radio" name="vista-equipo" :value="value" /><span>{{ value ? 'En equipo' : 'Tu día a día' }}</span></label></fieldset><div class="planLayout"><article v-for="(card,index) in roleCards" :key="card.name" class="planDetail" :class="{ featuredPlan:index===1 }" data-reveal><div class="planHeading"><h3>{{ card.name }}</h3><AppIcon v-if="index===1" name="flame" :size="20" /></div><p class="planDescription">{{ card.description }}</p><div class="planSummary" :aria-live="index===1 ? 'polite' : 'off'" aria-atomic="true"><p class="planPrice">{{ card.label }}</p><p>{{ card.period }}</p></div><router-link to="/login" :class="index===1 ? 'primary' : 'darkButton'">{{ card.cta }}<AppIcon name="arrow" :size="17" /></router-link><ul class="planFeatures"><li v-for="feature in card.features" :key="feature"><AppIcon name="check" :size="17" />{{ feature }}</li></ul></article></div><p class="planFoot">Propietario, administrador y empleado. Cada rol tiene su propio espacio de trabajo.</p></section>

  <section id="preguntas" class="faq" aria-labelledby="faq-title"><div class="faqIntro"><h2 id="faq-title"><span class="textMotion" data-text-motion>Antes de abrir la cocina.</span></h2><p data-reveal>Respuestas a las dudas más comunes para empezar a trabajar con LeguiBurger.</p></div><div class="faqList"><article v-for="(faq,index) in faqs" :key="faq.question" class="faqItem" :data-open="openFaq===index" data-reveal><h3><button :id="`faq-trigger-${index}`" type="button" :aria-expanded="openFaq===index" :aria-controls="`faq-answer-${index}`" @click="openFaq=openFaq===index ? null : index">{{ faq.question }}<AppIcon name="chevron" :size="22" /></button></h3><div :id="`faq-answer-${index}`" class="faqAnswer" role="region" :aria-labelledby="`faq-trigger-${index}`" :aria-hidden="openFaq!==index" :inert="openFaq!==index || undefined"><div><p>{{ faq.answer }}</p></div></div></article></div></section>
  <section class="finalCta" aria-labelledby="cta-title"><h2 id="cta-title"><span class="textMotion" data-text-motion>Tu próxima jornada,<br />con todo a punto.</span></h2><div data-reveal><p>Tu marca, tus locales y tu cocina. Todo en LeguiBurger.</p><router-link to="/login" class="darkButton">Entrar a mi negocio<AppIcon name="arrow" :size="18" /></router-link><a class="finalContact" href="#como-funciona">El sabor es tuyo. El orden, también.</a></div></section>
  <footer class="footer"><div><BrandLogo /><p>Menos administración. Más sabor.</p></div><nav aria-label="Enlaces del pie"><a href="#como-funciona">Cómo funciona</a><a href="#tu-espacio">Tu espacio</a><router-link to="/login">Ingresar</router-link></nav><p>LeguiBurger · Hecho para tu cocina.</p></footer>
 </main>
</template>

<style scoped src="../../styles/landing.css"></style>
