import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const loginDuration = new Trend('login_duration');
const signupDuration = new Trend('signup_duration');

// Test configuration - Simulates realistic load
export const options = {
  stages: [
    { duration: '2m', target: 20 },   // Ramp up to 20 users
    { duration: '5m', target: 50 },   // Ramp up to 50 users
    { duration: '10m', target: 50 },  // Stay at 50 users
    { duration: '3m', target: 100 },  // Spike to 100 users
    { duration: '5m', target: 50 },   // Scale back to 50
    { duration: '2m', target: 0 },    // Ramp down to 0
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.01'],
    errors: ['rate<0.01'],
    login_duration: ['p(95)<300'],
    signup_duration: ['p(95)<400'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_PATH = __ENV.API_PATH || '/api/auth';

// Shared state for users (30% new users, 70% returning users)
const users = [];

function generateEmail() {
  return `load-test-${__VU}-${Date.now()}-${Math.random().toString(36).substring(7)}@example.com`;
}

function generateName() {
  return `Load Test User ${__VU}`;
}

function getRandomUser() {
  if (users.length > 0 && Math.random() > 0.3) {
    return users[Math.floor(Math.random() * users.length)];
  }
  return null;
}

export default function () {
  const headers = { 'Content-Type': 'application/json' };
  
  // Decide: new user (30%) or returning user (70%)
  const existingUser = getRandomUser();
  
  if (!existingUser) {
    // New user flow
    group('User Registration', function () {
      const email = generateEmail();
      const password = 'LoadTest123!';
      const name = generateName();

      const payload = JSON.stringify({ email, password, name });
      const startTime = Date.now();
      
      const res = http.post(`${BASE_URL}${API_PATH}/register`, payload, { headers });
      signupDuration.add(Date.now() - startTime);

      const success = check(res, {
        'signup status is 201': (r) => r.status === 201,
        'signup has token': (r) => JSON.parse(r.body).token !== undefined,
      });

      if (success) {
        const body = JSON.parse(res.body);
        users.push({ email, password, token: body.token });
      } else {
        errorRate.add(1);
      }

      sleep(1);
    });
  } else {
    // Returning user flow
    group('User Login', function () {
      const payload = JSON.stringify({
        email: existingUser.email,
        password: existingUser.password,
      });
      
      const startTime = Date.now();
      const res = http.post(`${BASE_URL}${API_PATH}/login`, payload, { headers });
      loginDuration.add(Date.now() - startTime);

      const success = check(res, {
        'login status is 200': (r) => r.status === 200,
        'login has token': (r) => JSON.parse(r.body).token !== undefined,
      });

      if (success) {
        const body = JSON.parse(res.body);
        existingUser.token = body.token;
      } else {
        errorRate.add(1);
      }

      sleep(1);
    });

    // Authenticated operations
    if (existingUser.token) {
      const authHeaders = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${existingUser.token}`,
      };

      // Get user profile (50% of time)
      if (Math.random() > 0.5) {
        group('Get User Profile', function () {
          const res = http.get(`${BASE_URL}${API_PATH}/me`, { headers: authHeaders });
          
          check(res, {
            'profile status is 200': (r) => r.status === 200,
            'profile has email': (r) => JSON.parse(r.body).email === existingUser.email,
          }) || errorRate.add(1);

          sleep(0.5);
        });
      }

      // Update profile (10% of time)
      if (Math.random() > 0.9) {
        group('Update User Profile', function () {
          const payload = JSON.stringify({
            name: `Updated ${generateName()}`,
          });
          
          const res = http.put(`${BASE_URL}${API_PATH}/me`, payload, { headers: authHeaders });
          
          check(res, {
            'update status is 200': (r) => r.status === 200,
          }) || errorRate.add(1);

          sleep(0.5);
        });
      }

      // Logout (5% of time)
      if (Math.random() > 0.95) {
        group('User Logout', function () {
          const res = http.post(`${BASE_URL}${API_PATH}/logout`, '{}', { headers: authHeaders });
          
          check(res, {
            'logout status is 200': (r) => r.status === 200,
          }) || errorRate.add(1);

          // Remove token after logout
          existingUser.token = null;
        });
      }
    }

    sleep(Math.random() * 2 + 1); // Random think time 1-3 seconds
  }
}

export function handleSummary(data) {
  console.log('\n========================================');
  console.log('  AuthSome Load Test Summary');
  console.log('========================================\n');
  
  const checks = data.metrics.checks.values;
  const httpReqs = data.metrics.http_reqs.values;
  const httpDuration = data.metrics.http_req_duration.values;
  const httpFailed = data.metrics.http_req_failed.values;
  
  console.log(`Total Requests: ${httpReqs.count}`);
  console.log(`Request Rate: ${httpReqs.rate.toFixed(2)}/s`);
  console.log(`Failed Requests: ${(httpFailed.rate * 100).toFixed(2)}%`);
  console.log(`\nResponse Times:`);
  console.log(`  Average: ${httpDuration.avg.toFixed(2)}ms`);
  console.log(`  Median:  ${httpDuration.med.toFixed(2)}ms`);
  console.log(`  P95:     ${httpDuration['p(95)'].toFixed(2)}ms`);
  console.log(`  P99:     ${httpDuration['p(99)'].toFixed(2)}ms`);
  console.log(`  Max:     ${httpDuration.max.toFixed(2)}ms`);
  
  if (data.metrics.login_duration) {
    const loginDur = data.metrics.login_duration.values;
    console.log(`\nLogin Performance:`);
    console.log(`  Average: ${loginDur.avg.toFixed(2)}ms`);
    console.log(`  P95:     ${loginDur['p(95)'].toFixed(2)}ms`);
  }
  
  if (data.metrics.signup_duration) {
    const signupDur = data.metrics.signup_duration.values;
    console.log(`\nSignup Performance:`);
    console.log(`  Average: ${signupDur.avg.toFixed(2)}ms`);
    console.log(`  P95:     ${signupDur['p(95)'].toFixed(2)}ms`);
  }
  
  console.log(`\nChecks: ${checks.passes}/${checks.passes + checks.fails} passed\n`);
  console.log('========================================\n');
  
  return {
    'stdout': '\nDetailed results saved to results/load-test-summary.json\n',
    'results/load-test-summary.json': JSON.stringify(data, null, 2),
    'results/load-test-summary.html': generateHTML(data),
  };
}

function generateHTML(data) {
  return `
<!DOCTYPE html>
<html>
<head>
  <title>AuthSome Load Test Results</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
    .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
    h1 { color: #333; border-bottom: 3px solid #4CAF50; padding-bottom: 10px; }
    .metric { margin: 20px 0; padding: 15px; background: #f9f9f9; border-left: 4px solid #4CAF50; }
    .metric h3 { margin-top: 0; color: #555; }
    .value { font-size: 24px; font-weight: bold; color: #4CAF50; }
    .good { color: #4CAF50; }
    .warning { color: #FF9800; }
    .bad { color: #F44336; }
    table { width: 100%; border-collapse: collapse; margin: 20px 0; }
    th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
    th { background-color: #4CAF50; color: white; }
    tr:hover { background-color: #f5f5f5; }
  </style>
</head>
<body>
  <div class="container">
    <h1>AuthSome Load Test Results</h1>
    <p><strong>Generated:</strong> ${new Date().toISOString()}</p>
    
    <div class="metric">
      <h3>Total Requests</h3>
      <div class="value">${data.metrics.http_reqs.values.count}</div>
      <p>Rate: ${data.metrics.http_reqs.values.rate.toFixed(2)}/s</p>
    </div>
    
    <div class="metric">
      <h3>Response Times</h3>
      <table>
        <tr><th>Metric</th><th>Value</th></tr>
        <tr><td>Average</td><td>${data.metrics.http_req_duration.values.avg.toFixed(2)}ms</td></tr>
        <tr><td>Median</td><td>${data.metrics.http_req_duration.values.med.toFixed(2)}ms</td></tr>
        <tr><td>P95</td><td class="${data.metrics.http_req_duration.values['p(95)'] < 500 ? 'good' : 'warning'}">${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms</td></tr>
        <tr><td>P99</td><td class="${data.metrics.http_req_duration.values['p(99)'] < 1000 ? 'good' : 'warning'}">${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms</td></tr>
        <tr><td>Max</td><td>${data.metrics.http_req_duration.values.max.toFixed(2)}ms</td></tr>
      </table>
    </div>
    
    <div class="metric">
      <h3>Error Rate</h3>
      <div class="value ${data.metrics.http_req_failed.values.rate < 0.01 ? 'good' : 'bad'}">
        ${(data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%
      </div>
    </div>
    
    <div class="metric">
      <h3>Checks</h3>
      <div class="value">
        ${data.metrics.checks.values.passes}/${data.metrics.checks.values.passes + data.metrics.checks.values.fails} passed
      </div>
    </div>
  </div>
</body>
</html>
`;
}

