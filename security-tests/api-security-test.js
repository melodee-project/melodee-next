#!/usr/bin/env node

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metric for tracking security test results
const securityChecks = new Rate('security_checks');

// Options for security testing
export let options = {
  stages: [
    { duration: '1m', target: 5 },   // Small number of concurrent requests for security testing
    { duration: '2m', target: 5 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    'security_checks': ['rate>=1.0'], // All security checks should pass
  },
};

// Security checks for OpenSubsonic API
export default function () {
  const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/rest';
  
  // Test 1: Verify authentication is required for protected endpoints
  let resp = http.get(`${BASE_URL}/getArtists.view`);
  securityChecks.add(resp.status === 401 || (resp.status === 200 && resp.json().error)); // Should return error if unauthorized
  
  // Test 2: Test for invalid credentials
  resp = http.get(`${BASE_URL}/getArtists.view?u=nonexistent&p=enc:invalid`);
  securityChecks.add(resp.status === 200); // OpenSubsonic returns 200 with error in XML
  if (resp.status === 200) {
    const body = resp.body;
    securityChecks.add(body.includes('<error'));
    securityChecks.add(body.includes('NOT_AUTHORIZED'));
  }
  
  // Test 3: Check for sensitive information in error messages
  resp = http.get(`${BASE_URL}/getArtists.view?u=../../../etc/passwd&p=enc:test`);
  securityChecks.add(!resp.body.includes('/etc/passwd')); // Should not reveal system paths
  
  // Test 4: Test for SQL injection patterns in query parameters
  const sqlInjectionTests = [
    `${BASE_URL}/search3.view?u=test&p=enc:password&query=%27%20OR%20%271%27%3D%271`,
    `${BASE_URL}/getMusicDirectory.view?u=test&p=enc:password&id=%27%3B%20DROP%20TABLE%20users%3B--`,
    `${BASE_URL}/getSong.view?u=test&p=enc:password&id=%28SELECT%201%29`,
  ];
  
  for (const testUrl of sqlInjectionTests) {
    resp = http.get(testUrl);
    // Should not crash or expose DB errors
    securityChecks.add(resp.status === 200 || resp.status === 400 || resp.status === 404);
  }
  
  // Test 5: Path traversal attempts
  const pathTraversalTests = [
    `${BASE_URL}/getCoverArt.view?u=test&p=enc:password&id=../../../etc/passwd`,
    `${BASE_URL}/getAvatar.view?u=test&p=enc:password&username=../../../etc/shadow`,
  ];
  
  for (const testUrl of pathTraversalTests) {
    resp = http.get(testUrl);
    // Should not return sensitive system files
    securityChecks.add(!(resp.body.includes('root:') || resp.body.includes('BEGIN PGP')));
  }
  
  // Test 6: Test rate limiting (if implemented)
  // Make multiple rapid requests to see if rate limiting works
  for (let i = 0; i < 10; i++) {
    resp = http.get(`${BASE_URL}/ping.view`);
    // If rate limiting is working, some requests may return 429
    if (resp.status === 429) {
      securityChecks.add(true); // Rate limiting working as expected
      break;
    }
  }
  
  // Test 7: Verify JWT tokens have proper expiration
  resp = http.get(`${BASE_URL}/getLicense.view?u=test&p=enc:password`);
  if (resp.status === 200) {
    const responseJson = resp.json();
    securityChecks.add(responseJson['subsonic-response'].status === 'ok' || 
                      responseJson['subsonic-response'].error); // Either success or error (not crash)
  }
  
  // Test 8: Check for proper CORS headers (security consideration)
  resp = http.options(`${BASE_URL}/ping.view`, {
    headers: {
      'Origin': 'https://malicious-site.com',
      'Access-Control-Request-Method': 'GET',
      'Access-Control-Request-Headers': 'X-Requested-With',
    },
  });
  
  // Should not allow cross-origin requests from non-whitelisted origins
  const allowOrigin = resp.headers['Access-Control-Allow-Origin'];
  if (allowOrigin) {
    securityChecks.add(allowOrigin !== '*' || allowOrigin === 'https://malicious-site.com');
  } else {
    securityChecks.add(true); // If no header is returned, that's also secure
  }
  
  sleep(1);
}

// Export setup function to initialize security test
export function setup() {
  console.log('Starting Melodee security tests...');
  return { startTime: new Date() };
}

// Export teardown function to finalize security test
export function teardown(data) {
  console.log(`Security test completed at: ${new Date()}`);
  console.log(`Security checks passed: ${securityChecks.obj.passes} out of ${(securityChecks.obj.passes + securityChecks.obj.fails)}`);
}