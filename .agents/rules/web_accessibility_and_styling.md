# Web Development, Accessibility & Design System Rules

These rules apply whenever viewing, editing, or generating code within the `web/` frontend directory.

## 1. Technology Stack
- **Framework**: Vue 3 with Composition API (`<script setup lang="ts">`).
- **State Management**: Pinia stores in `web/src/stores/`.
- **Router**: Vue Router in `web/src/router/`.
- **Localization**: `vue-i18n` with Italian (`it`) and English (`en`) in `web/src/locales/`.
- **Icons**: `@lucide/vue`.
- **Styling**: Vanilla CSS Design System with custom CSS variables in `tokens.css`.

## 2. Accessibility (a11y) Requirements
1. **Semantic HTML & ARIA Attributes**:
   - Use semantic elements (`<main>`, `<header>`, `<nav>`, `<article>`, `<section>`, `<button>`).
   - Every `<button>` must have visible text or an `aria-label`.
   - Toggle buttons must declare `role="switch"` and `:aria-checked="isOn"`.
   - Dialogs/Modals must have `role="dialog"`, `aria-modal="true"`, and an `aria-labelledby` pointing to the title.
2. **Keyboard Navigation**:
   - Visible focus indicator: Never set `outline: none` without providing an active `:focus-visible` replacement.
   - Interactive cards must respond to both `Enter` and `Space` key presses.
   - Include a skip navigation link (`<a href="#main-content" class="skip-link">Skip to Content</a>`).
3. **Motion Sensitivity**:
   - Wrap non-essential animations and transitions in `@media (prefers-reduced-motion: reduce)`.
4. **Color Contrast & Readability**:
   - Minimum text contrast ratio of **4.5:1** for regular text and **3:1** for large text against card/panel backgrounds.
   - Never disable text selection on diagnostic data, entity IDs, logs, or user inputs.

## 3. Visual Aesthetics & Design System
- **Dark Mode First**: Deep slate background (`--bg-primary: #0b1120`) with cyan accent (`--primary: #0ea5e9`).
- **Glassmorphism**: Use `.glass-panel` and `.glass-card` classes with `backdrop-filter: blur(12px)`.
- **Responsive Layout**: Fluid CSS Grid and Flexbox with mobile-first breakpoints (`min-width: 640px`, `1024px`, `1280px`).
- **Micro-Animations**: Smooth hover elevations (`transform: translateY(-2px)`), active press scales (`transform: scale(0.98)`), and spring transitions.
