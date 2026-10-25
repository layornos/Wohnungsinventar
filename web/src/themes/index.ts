// Loads every theme's CSS file (lib/themes.ts lists the themes). A module of its own, because the imports a glob
// turns into are hoisted to the top of the importing module, which would put the themes before app.css.
import.meta.glob('./*.css', { eager: true });
