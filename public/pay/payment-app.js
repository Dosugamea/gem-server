// Payment App JavaScript
// 決済アプリウィンドウのロジック

let paymentRequestClient = null;
let paymentRequestData = null;
let userBalance = null;
let authToken = null;

// APIベースURL（環境に応じて変更）
const API_BASE_URL = window.location.origin + '/api/v1';

// 初期化
function init() {
  console.log('[Payment App] Initializing...');
  
  // Service Workerからのメッセージを受信
  if (navigator.serviceWorker) {
    navigator.serviceWorker.addEventListener('message', handleServiceWorkerMessage);
    
    // Service Workerが制御しているか確認
    if (navigator.serviceWorker.controller) {
      // Service Workerに準備完了を通知
      navigator.serviceWorker.controller.postMessage('payment_app_window_ready');
      console.log('[Payment App] Sent ready message to service worker');
    } else {
      console.error('[Payment App] Service Worker controller not found');
      showError('Service Workerが利用できません');
    }
  } else {
    console.error('[Payment App] Service Worker not supported');
    showError('Service Workerがサポートされていません');
  }
  
  // ボタンイベントの設定
  document.getElementById('approve-btn').addEventListener('click', handlePaymentApproval);
  document.getElementById('cancel-btn').addEventListener('click', handleCancel);
  
  // 認証トークンの取得（ローカルストレージまたはセッションから）
  authToken = getAuthToken();
  if (!authToken) {
    showError('認証トークンが見つかりません。ログインしてください。');
    return;
  }
}

// Service Workerからのメッセージ処理
function handleServiceWorkerMessage(e) {
  console.log('[Payment App] Message from service worker:', e.data);
  
  paymentRequestClient = e.source;
  
  if (e.data.type === 'payment_request_data') {
    // 決済情報を受信
    paymentRequestData = {
      total: e.data.total,
      methodData: e.data.methodData,
      modifiers: e.data.modifiers
    };
    
    displayPaymentInfo(paymentRequestData);
    loadUserBalance(); // ユーザー残高を取得
  }
}

// 決済情報の表示
function displayPaymentInfo(data) {
  const itemName = data.total?.label || '商品';
  const amount = data.total?.amount?.value || '0';
  const currency = data.total?.amount?.currency || 'JPY';
  
  document.getElementById('item-name').textContent = itemName;
  document.getElementById('total-amount').textContent = formatCurrency(amount, currency);
  
  // ローディングを非表示にしてコンテンツを表示
  document.getElementById('loading').style.display = 'none';
  document.getElementById('payment-content').style.display = 'block';
}

// ユーザー残高の取得
async function loadUserBalance() {
  try {
    // バックエンドAPIから残高を取得（ユーザーAPI: /me/balance）
    const response = await fetch(`${API_BASE_URL}/me/balance`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json'
      }
    });
    
    if (!response.ok) {
      if (response.status === 401) {
        throw new Error('認証に失敗しました。再度ログインしてください。');
      }
      throw new Error(`残高の取得に失敗しました: ${response.status}`);
    }
    
    const data = await response.json();
    userBalance = data;
    
    displayBalance(data);
    
    // 購入金額を取得
    const totalAmount = parseFloat(paymentRequestData.total.amount.value);
    
    // 残高チェック（有償通貨と無料通貨の合計）
    const totalBalance = parseFloat(data.balances?.paid || 0) + parseFloat(data.balances?.free || 0);
    
    if (totalBalance < totalAmount) {
      showInsufficientBalanceWarning();
      document.getElementById('approve-btn').disabled = true;
    } else {
      document.getElementById('approve-btn').disabled = false;
    }
  } catch (error) {
    console.error('[Payment App] Failed to load balance:', error);
    showError(error.message || '残高情報の取得に失敗しました');
    document.getElementById('approve-btn').disabled = true;
  }
}

// 残高の表示
function displayBalance(balance) {
  const paidBalance = parseFloat(balance.balances?.paid || 0);
  const freeBalance = parseFloat(balance.balances?.free || 0);
  const totalBalance = paidBalance + freeBalance;
  
  document.getElementById('total-balance').textContent = formatCurrency(totalBalance.toString(), 'JPY');
  document.getElementById('paid-balance').textContent = formatCurrency(paidBalance.toString(), 'JPY');
  document.getElementById('free-balance').textContent = formatCurrency(freeBalance.toString(), 'JPY');
}

// 残高不足の警告表示
function showInsufficientBalanceWarning() {
  document.getElementById('insufficient-balance').style.display = 'block';
}

// エラーメッセージの表示
function showError(message) {
  const errorElement = document.getElementById('error-message');
  errorElement.textContent = message;
  errorElement.classList.add('show');
  
  // ローディングを非表示
  document.getElementById('loading').style.display = 'none';
  document.getElementById('payment-content').style.display = 'block';
}

// 決済承認処理
async function handlePaymentApproval() {
  if (!paymentRequestClient) {
    showError('Service Workerとの接続が確立されていません');
    return;
  }
  
  if (!authToken) {
    showError('認証トークンが見つかりません');
    return;
  }
  
  try {
    // ボタンを無効化
    document.getElementById('approve-btn').disabled = true;
    document.getElementById('approve-btn').textContent = '処理中...';
    
    // ユーザーIDを取得
    const userId = getUserIdFromToken(authToken);
    if (!userId) {
      throw new Error('ユーザーIDを取得できませんでした');
    }
    
    // トランザクションIDを生成
    const transactionId = generateTransactionId();
    
    // PaymentResponseを作成
    // 決済方法のURLは現在のオリジンを使用
    const methodName = window.location.origin + '/pay';
    const paymentResponse = {
      methodName: methodName,
      details: {
        userId: userId,
        transactionId: transactionId,
        timestamp: Date.now(),
        // セキュリティのため、トークンは含めない（バックエンドで検証）
      }
    };
    
    console.log('[Payment App] Sending payment response:', paymentResponse);
    
    // Service Workerに送信
    paymentRequestClient.postMessage(paymentResponse);
    
    // ウィンドウを閉じる（Service Workerが処理を完了するまで待つ）
    setTimeout(() => {
      window.close();
    }, 500);
    
  } catch (error) {
    console.error('[Payment App] Payment approval error:', error);
    showError(error.message || '決済処理中にエラーが発生しました');
    document.getElementById('approve-btn').disabled = false;
    document.getElementById('approve-btn').textContent = '決済を承認';
    
    // エラーをService Workerに通知
    if (paymentRequestClient) {
      paymentRequestClient.postMessage({
        type: 'payment_error',
        error: error.message
      });
    }
  }
}

// キャンセル処理
function handleCancel() {
  if (paymentRequestClient) {
    paymentRequestClient.postMessage({
      type: 'payment_cancelled'
    });
  }
  window.close();
}

// 認証トークンの取得（実装に応じて変更）
function getAuthToken() {
  // ローカルストレージから取得
  const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
  
  // URLパラメータから取得（開発用）
  if (!token) {
    const params = new URLSearchParams(window.location.search);
    return params.get('token');
  }
  
  return token;
}

// トークンからユーザーIDを取得（実装に応じて変更）
function getUserIdFromToken(token) {
  if (!token) return null;
  
  try {
    // JWTトークンのペイロードをデコード（簡易実装）
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    
    const payload = JSON.parse(atob(parts[1]));
    return payload.sub || payload.user_id || payload.userId;
  } catch (error) {
    console.error('[Payment App] Failed to decode token:', error);
    return null;
  }
}

// トランザクションIDの生成
function generateTransactionId() {
  return 'txn_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
}

// 通貨フォーマット
function formatCurrency(amount, currency) {
  const num = parseFloat(amount);
  if (isNaN(num)) return amount;
  
  if (currency === 'JPY') {
    return '¥' + Math.floor(num).toLocaleString('ja-JP');
  } else {
    return new Intl.NumberFormat('ja-JP', {
      style: 'currency',
      currency: currency
    }).format(num);
  }
}

// ページ読み込み時に初期化
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}
