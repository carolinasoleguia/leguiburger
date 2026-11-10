export function mountLandingWheel(root) {
    const landing = root;
    if (!landing) return;
    let locked = false;
    let animationFrame;

    const handleWheel = (event) => {
      if (event.ctrlKey) return;
      const wheelDelta = event.deltaY * (event.deltaMode === WheelEvent.DOM_DELTA_LINE ? 16 : event.deltaMode === WheelEvent.DOM_DELTA_PAGE ? window.innerHeight : 1);
      if (locked) {
        event.preventDefault();
        return;
      }
      if (Math.abs(wheelDelta) < 8) return;
      const target = event.target;
      if (!(target instanceof Element) || target.closest('[role="dialog"], input, textarea, select, [contenteditable="true"]')) return;

      let ancestor = target.parentElement;
      while (ancestor && ancestor !== landing) {
        const { overflowY } = window.getComputedStyle(ancestor);
        if ((overflowY === 'auto' || overflowY === 'scroll') && ancestor.scrollHeight > ancestor.clientHeight + 1) {
          const canScroll = wheelDelta > 0
            ? ancestor.scrollTop + ancestor.clientHeight < ancestor.scrollHeight - 1
            : ancestor.scrollTop > 0;
          if (canScroll) return;
        }
        ancestor = ancestor.parentElement;
      }

      const scrollY = window.scrollY;
      // Keep scroll stops directly below the sticky header at every width.
      // Measuring the rendered header avoids different jumps when resizing
      // around CSS breakpoints (including laptop-sized viewports).
      const header = landing.querySelector('header');
      const anchorOffset = Math.ceil(header?.getBoundingClientRect().height ?? 82);
      const viewportStep = Math.max(1, window.innerHeight - anchorOffset);
      const anchors = [];
      const addAnchor = (element, offset) => {
        const position = element.getBoundingClientRect().top + scrollY - anchorOffset + offset;
        anchors.push({ element, offset, position: Math.max(0, position) });
      };
      const sections = landing.querySelectorAll(':scope > section');
      for (const section of sections) {
        if (section.id === 'tu-espacio') {
          addAnchor(section, 0);
          const panelLayout = section.querySelector('[data-scroll-stop="panel-layout"]');
          const integrations = section.querySelector('[data-scroll-stop="integrations"]');
          // The panel stays with its heading on laptop and desktop layouts.
          // On narrow layouts it becomes a separate stop after the intro.
          if (panelLayout && window.innerWidth <= 850) addAnchor(panelLayout, 0);
          if (integrations) {
            const panelExample = section.querySelector('[data-ui="panel-example"]');
            const integrationHeading = integrations.querySelector('h2');
            // Preserve the product preview only when the integration heading
            // also fits below it. Otherwise make that heading the next clear stop.
            const previewToHeading = panelExample && integrationHeading
              ? integrationHeading.getBoundingClientRect().bottom - panelExample.getBoundingClientRect().top
              : Number.POSITIVE_INFINITY;
            const keepExampleVisible = window.innerWidth > 1440
              && panelExample !== null
              && anchorOffset + previewToHeading <= window.innerHeight - 24;
            const offset = keepExampleVisible
              ? panelExample.getBoundingClientRect().top - integrations.getBoundingClientRect().top
              : 0;
            addAnchor(integrations, offset);
          }
          continue;
        }
        const overflow = Math.max(0, section.offsetHeight - viewportStep);
        const minimumUsefulStep = viewportStep * .4;
        if (overflow >= minimumUsefulStep) {
          const pageCount = Math.ceil(overflow / viewportStep);
          const step = overflow / pageCount;
          for (let page = 0; page <= pageCount; page++) {
            addAnchor(section, Math.round(step * page));
          }
        } else {
          addAnchor(section, 0);
        }
      }
      const footer = landing.querySelector(':scope > footer');
      if (footer) addAnchor(footer, 0);
      anchors.sort((left, right) => left.position - right.position);

      const nextAnchor = wheelDelta > 0
        ? anchors.find(anchor => anchor.position > scrollY + 28)
        : [...anchors].reverse().find(anchor => anchor.position < scrollY - 28);
      if (nextAnchor === undefined) return;

      event.preventDefault();
      locked = true;
      const distance = nextAnchor.element.getBoundingClientRect().top + nextAnchor.offset - anchorOffset;
      const duration = wheelDelta < 0 ? 300 : 750;
      let startedAt;
      let previousEased = 0;
      const animate = (now) => {
        startedAt ??= now;
        const progress = Math.min(1, (now - startedAt) / duration);
        const eased = (1 - Math.cos(Math.PI * progress)) / 2;
        window.scrollBy(0, distance * (eased - previousEased));
        previousEased = eased;
        if (progress < 1) {
          animationFrame = requestAnimationFrame(animate);
        } else {
          const correction = nextAnchor.element.getBoundingClientRect().top + nextAnchor.offset - anchorOffset;
          if (Math.abs(correction) > 1) window.scrollBy(0, correction);
          animationFrame = undefined;
          locked = false;
        }
      };
      animationFrame = requestAnimationFrame(animate);
    };

    landing.addEventListener('wheel', handleWheel, { passive: false });
    return () => {
      landing.removeEventListener('wheel', handleWheel);
      if (animationFrame !== undefined) cancelAnimationFrame(animationFrame);
    };
}
