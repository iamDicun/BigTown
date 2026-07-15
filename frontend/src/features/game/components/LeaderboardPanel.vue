<template>
  <section class="panel leaderboard-panel" aria-label="Leaderboard">
    <header>Leaderboard</header>
    <ol>
      <li>
        <span>Chưa có dữ liệu</span>
        <strong>0</strong>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.panel {
  border: 1px solid rgba(159, 212, 127, 0.24);
  border-radius: 14px;
  background: rgba(16, 22, 16, 0.86);
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.28);
  backdrop-filter: blur(12px);
}

header {
  padding: 12px 14px;
  border-bottom: 1px solid rgba(159, 212, 127, 0.18);
  font-weight: 700;
}

ol {
  margin: 0;
  padding: 12px 14px;
  list-style-position: inside;
}

li {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  color: #c9c1aa;
}
</style>
