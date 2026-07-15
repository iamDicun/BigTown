<template>
  <section class="panel chat-panel" aria-label="Game chat">
    <header>Chat</header>
    <div class="messages">
      <p>Chat WebSocket sẽ được nối vào đây.</p>
    </div>
    <form class="chat-form">
      <input type="text" placeholder="Nhắn trong map..." disabled>
      <button type="button" disabled>Gửi</button>
    </form>
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

.chat-panel {
  min-height: 0;
  display: grid;
  grid-template-rows: auto 1fr auto;
}

header {
  padding: 12px 14px;
  border-bottom: 1px solid rgba(159, 212, 127, 0.18);
  font-weight: 700;
}

.messages {
  min-height: 120px;
  padding: 14px;
  color: #c9c1aa;
}

.chat-form {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px;
  padding: 12px;
}

input,
button {
  min-width: 0;
  border: 1px solid #566744;
  border-radius: 8px;
  padding: 9px 10px;
  background: #172017;
  color: #f3ead7;
}
</style>
