// Payment Handler Service Worker
// 決済リクエストを処理するService Worker

// 現在の決済リクエストを保持
let currentPayment = {
  paymentRequestEvent: null,
  resolver: null
};

// PromiseResolver クラス（PaymentRequestEventのPromiseを解決するため）
class PromiseResolver {
  constructor() {
    this.promise = new Promise((resolve, reject) => {
      this.resolve = resolve;
      this.reject = reject;
    });
  }
}

// canmakepaymentイベント: 決済可能かどうかの確認
self.addEventListener('canmakepayment', (e) => {
  // 本システムでは常にtrueを返す（ユーザー認証は決済ウィンドウで行う）
  e.respondWith(true);
});

// paymentrequestイベント: 決済リクエストの処理
self.addEventListener('paymentrequest', async (e) => {
  console.log('[Service Worker] Payment request received:', e.methodData);
  
  // イベントとPromiseリゾルバを保存
  currentPayment.paymentRequestEvent = e;
  currentPayment.resolver = new PromiseResolver();
  
  // Promiseを返す（後でresolve/rejectされる）
  e.respondWith(currentPayment.resolver.promise);
  
  // 決済ウィンドウを開く
  try {
    const windowClient = await e.openWindow('/pay/index.html');
    if (windowClient === null) {
      console.error('[Service Worker] Failed to open payment window');
      currentPayment.resolver.reject('Failed to open payment window');
    } else {
      console.log('[Service Worker] Payment window opened successfully');
    }
  } catch (err) {
    console.error('[Service Worker] Error opening payment window:', err);
    currentPayment.resolver.reject(err);
  }
});

// 決済ウィンドウからのメッセージ受信
self.addEventListener('message', (e) => {
  console.log('[Service Worker] Message received:', e.data);
  
  if (!currentPayment.resolver) {
    console.warn('[Service Worker] No active payment request, ignoring message');
    return; // 決済リクエストがない場合は無視
  }
  
  // 決済ウィンドウが準備完了
  if (e.data === 'payment_app_window_ready') {
    console.log('[Service Worker] Payment app window ready, sending payment request data');
    // 決済情報をウィンドウに送信
    sendPaymentRequestToClient(e.source);
    return;
  }
  
  // 決済承認
  if (e.data.methodName) {
    console.log('[Service Worker] Payment approved:', e.data.methodName);
    // PaymentResponseを返す
    currentPayment.resolver.resolve({
      methodName: e.data.methodName,
      details: e.data.details
    });
    // リセット
    currentPayment.paymentRequestEvent = null;
    currentPayment.resolver = null;
  } else if (e.data === 'payment_cancelled' || e.data.type === 'payment_cancelled') {
    // キャンセル
    console.log('[Service Worker] Payment cancelled by user');
    currentPayment.resolver.reject('Payment cancelled by user');
    // リセット
    currentPayment.paymentRequestEvent = null;
    currentPayment.resolver = null;
  } else if (e.data.type === 'payment_error') {
    // エラー
    console.error('[Service Worker] Payment error:', e.data.error);
    currentPayment.resolver.reject(e.data.error || 'Payment error occurred');
    // リセット
    currentPayment.paymentRequestEvent = null;
    currentPayment.resolver = null;
  }
});

// 決済情報をウィンドウに送信
function sendPaymentRequestToClient(client) {
  if (!currentPayment.paymentRequestEvent) {
    console.warn('[Service Worker] No payment request event to send');
    return;
  }
  
  const paymentData = {
    total: currentPayment.paymentRequestEvent.total,
    methodData: currentPayment.paymentRequestEvent.methodData,
    modifiers: currentPayment.paymentRequestEvent.modifiers || []
  };
  
  console.log('[Service Worker] Sending payment request data to window:', paymentData);
  client.postMessage({
    type: 'payment_request_data',
    ...paymentData
  });
}

// インストール時の処理
self.addEventListener('install', (e) => {
  console.log('[Service Worker] Installing payment handler');
  self.skipWaiting(); // すぐにアクティブにする
});

// アクティベート時の処理
self.addEventListener('activate', (e) => {
  console.log('[Service Worker] Activating payment handler');
  e.waitUntil(clients.claim()); // すぐにクライアントを制御する
});
