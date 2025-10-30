import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },  // Ramp up to 10 users
    { duration: '1m', target: 10 },   // Stay at 10 users
    { duration: '30s', target: 0 },   // Ramp down to 0
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% < 500ms, 99% < 1s
    http_req_failed: ['rate<0.01'],                  // Less than 1% errors
    errors: ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_PATH = __ENV.API_PATH || '/api/auth';

// Generate unique email for each VU
function generateEmail() {
  return `test-${__VU}-${Date.now()}@example.com`;
}

// Generate random name
function generateName() {
  return `Test User ${__VU}-${__ITER}`;
}

export default function () {
  const email = generateEmail();
  const password = 'SecureTestPass123!';
  const name = generateName();

  // 1. User Registration
  const signupPayload = JSON.stringify({
    email: email,
    password: password,
    name: name,
  });

  const signupHeaders = {
    'Content-Type': 'application/json',
  };

  const signupRes = http.post(
    `${BASE_URL}${API_PATH}/register`,
    signupPayload,
    { headers: signupHeaders }
  );

  check(signupRes, {
    'signup status is 201': (r) => r.status === 201,
    'signup has user data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.user && body.user.email === email;
      } catch (e) {
        return false;
      }
    },
    'signup has session token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.token && body.token.length > 0;
      } catch (e) {
        return false;
      }
    },
  }) || errorRate.add(1);

  if (signupRes.status !== 201) {
    console.error(`Signup failed: ${signupRes.status} - ${signupRes.body}`);
    return;
  }

  const signupBody = JSON.parse(signupRes.body);
  const token = signupBody.token;

  sleep(1);

  // 2. Verify Session (Get Current User)
  const sessionHeaders = {
    'Authorization': `Bearer ${token}`,
  };

  const meRes = http.get(
    `${BASE_URL}${API_PATH}/me`,
    { headers: sessionHeaders }
  );

  check(meRes, {
    'get user status is 200': (r) => r.status === 200,
    'get user returns correct email': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.email === email;
      } catch (e) {
        return false;
      }
    },
  }) || errorRate.add(1);

  sleep(1);

  // 3. User Logout
  const logoutRes = http.post(
    `${BASE_URL}${API_PATH}/logout`,
    '{}',
    { headers: sessionHeaders }
  );

  check(logoutRes, {
    'logout status is 200': (r) => r.status === 200,
  }) || errorRate.add(1);

  sleep(1);

  // 4. User Login (with previous credentials)
  const loginPayload = JSON.stringify({
    email: email,
    password: password,
  });

  const loginRes = http.post(
    `${BASE_URL}${API_PATH}/login`,
    loginPayload,
    { headers: signupHeaders }
  );

  check(loginRes, {
    'login status is 200': (r) => r.status === 200,
    'login has user data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.user && body.user.email === email;
      } catch (e) {
        return false;
      }
    },
    'login has session token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.token && body.token.length > 0;
      } catch (e) {
        return false;
      }
    },
  }) || errorRate.add(1);

  sleep(2);
}

export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'results/auth-flow-summary.json': JSON.stringify(data),
  };
}

// Simple text summary function
function textSummary(data, options = {}) {
  const indent = options.indent || '';
  const enableColors = options.enableColors !== false;
  
  let output = '\n';
  output += indent + '✓ checks.........................: ' + formatPercentage(data.metrics.checks.values.passes, data.metrics.checks.values.fails) + '\n';
  output += indent + '✓ data_received.................: ' + formatBytes(data.metrics.data_received.values.count) + '\n';
  output += indent + '✓ data_sent.....................: ' + formatBytes(data.metrics.data_sent.values.count) + '\n';
  output += indent + '✓ http_req_duration.............: avg=' + formatDuration(data.metrics.http_req_duration.values.avg) + 
            ' p(95)=' + formatDuration(data.metrics.http_req_duration.values['p(95)']) + '\n';
  output += indent + '✓ http_req_failed...............: ' + formatPercentage(data.metrics.http_req_failed.values.rate * data.metrics.http_reqs.values.count, data.metrics.http_reqs.values.count) + '\n';
  output += indent + '✓ http_reqs.....................: ' + data.metrics.http_reqs.values.count + '\n';
  output += indent + '✓ iteration_duration............: avg=' + formatDuration(data.metrics.iteration_duration.values.avg) + '\n';
  output += indent + '✓ iterations....................: ' + data.metrics.iterations.values.count + '\n';
  output += indent + '✓ vus...........................: ' + data.metrics.vus.values.value + '\n';
  
  return output;
}

function formatDuration(ms) {
  if (ms < 1) return (ms * 1000).toFixed(2) + 'µs';
  if (ms < 1000) return ms.toFixed(2) + 'ms';
  return (ms / 1000).toFixed(2) + 's';
}

function formatBytes(bytes) {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
}

function formatPercentage(numerator, denominator) {
  if (denominator === 0) return '0.00%';
  return ((numerator / (numerator + denominator)) * 100).toFixed(2) + '%';
}

