<template>
  <main style='margin-right: 1rem'>
    <h2>Material Elevation Generator</h2>
    <div class='controlHolder'>
      <label for="dk-mode">Dark mode</label>
      <input type="checkbox" name="Dark mode" id="dk-mode">
      <label for="elev">Elevation (0-40)</label>
      <input type="range" min="0" max="40" value="0" class="slider" id="elev">
    </div>
    <div class='card' id='demo'>
      <h1 id='curElev'>0</h1>
    </div>
  </main>
  <main>
    <h2>Generated Code</h2>
    <div class='br'></div>
    <p id='cTitle'>Loading...</p>
    <code id='code'>Please wait</code>
  </main>
</template>

<style lang="scss" >
div.card {
  width: 250px;
  height: 250px;
  margin: 0 auto;

  background-color: #121212;
  background-image: linear-gradient(rgba(255, 255, 255, 0), rgba(255, 255, 255, 0));
  border-radius: 7px;

  box-shadow: 0px 2px 1px -1px rgba(0, 0, 0, 0.2), 0px 1px 1px 0px rgba(0, 0, 0, 0.14), 0px 1px 3px 0px rgba(0, 0, 0, 0.12);

  transition: background-color .25s ease-in-out, box-shadow .1s ease-in-out;
}

body.light div.card {
  background-color: #fff;
}

/* Styling for UI */

input[type=range] {
  -webkit-appearance: none;
  background: transparent;
}

input[type=range]::-webkit-slider-runnable-track {
  border-radius: 3px;
  height: 6px;
  background-color: #555
}

input[type=range]::-webkit-slider-thumb {
  -webkit-appearance: none;
  border: none;
  height: 16px;
  width: 16px;
  border-radius: 50%;
  background: #1976d2;
  margin-top: -5px;
}

h2 {
  margin-top: 0;
  margin-bottom: 1rem;
  text-align: center;
}

p {
  margin: .75rem 0 .25rem;
}

div.br {
  width: 100%;
  height: 1px;
  background-color: var(--border);
  margin-top: -.25rem;
}

code {
  font-family: monospace, Courier;
  white-space: pre-line;

  display: block;
  padding: .25rem .5rem;

  border: 1px solid var(--border);
  border-radius: 4px;
}

div.controlHolder {
  display: flex;
  justify-content: center;
  margin-bottom: 1rem;
}

div.controlHolder>*:not(label) {
  margin-right: 1rem;
}

div.controlHolder>*:last-child {
  margin-right: 0;
}

div.controlHolder>label {
  margin-right: .25rem;
}

* {
  font-family: Roboto, sans-serif;
  color: var(--txt-col);
  transition: color .25s ease-in-out;
}

body {
  background-color: var(--bg-col);
  margin: 0;
  padding: 0;
  min-height: 100vh;
  transition: background-color .25s ease-in-out;

  --txt-col: #eee;
  --base-bg: #000;
  --bg-col: #323232;
  --border: #444;
}

body.light {
  --bg-col: #eee;
  --txt-col: #121212;
  --base-bg: #fff;
  --border: #bbb;
}

body,
div.card {
  display: flex;
  justify-content: center;
  align-items: center;
}

main {
  padding: 1.5rem;
  border-radius: 7px;
  background-color: var(--base-bg);
  transition: background-color .25s ease-in-out;
  width: 410px;

  box-shadow: 0px 2px 3px -1px rgba(0, 0, 0, 0.2), 0px 4px 6px 1px rgba(0, 0, 0, 0.14), 0px 2px 8px 1px rgba(0, 0, 0, 0.12);
}
</style>

<script>
const r4 = (v) => v.toFixed(3);

export default {
  data() {
    return {
      elem: '',
      elev: '',
      label: '',
      cTitle: '',
      code: '',
      dkMode: false,
    };
  },
  mounted() {

    this.elem = document.getElementById('demo');
    this.label = document.getElementById('curElev');

    this.cTitle = document.getElementById('cTitle');
    this.code = document.getElementById('code');

    this.dkMode = localStorage.theme === 'd';
    this.elev = document.getElementById('elev');
    this.elev.oninput = this.updateElev;

    const t = document.getElementById('dk-mode');
    t.checked = this.dkMode;
    t.onchange = this.syncTheme;

    this.syncTheme({ target: t });
  },
  methods: {
    updateElev(e) {
      const v = e.target.value;
      this.label.textContent = v;

      if (this.dkMode) {
        const o = r4(v * 0.006666);
        const c = `linear-gradient(rgba(255, 255, 255, ${o}), rgba(255, 255, 255, ${o}))`;
        this.elem.style.backgroundImage = c;

        this.cTitle.textContent = 'background-image & background-color';
        this.code.textContent = `background-color: #121212; background-image: ${c};`;
        return;
      }

      const c = `0px ${r4(v * .4583)}px ${r4(v * .645)}px ${r4(v * -.2916)}px rgba(0,0,0,0.2), 0px ${v}px ${r4(v * 1.583)}px ${r4(v * .125)}px rgba(0,0,0,0.14), 0px ${r4(v * .375)}px ${r4(v * 1.916)}px ${r4(v * .3333)}px rgba(0,0,0,0.12)`;
      this.elem.style.boxShadow = c;

      this.cTitle.textContent = 'box-shadow';
      this.code.textContent = `box-shadow: ${c};`;

    },
    syncTheme(e) {
      this.dkMode = e.target.checked;
      this.dkMode
        ? document.body.classList.remove('light')
        : document.body.classList.add('light');
      localStorage.theme = this.dkMode ? 'd' : 'l';

      this.elem.style.boxShadow = null;
      this.elem.style.backgroundImage = null;
      this.updateElev({ target: this.elev });
    }
  }
}
</script>