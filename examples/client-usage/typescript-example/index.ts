/**
 * TypeScript Client Example
 * 
 * This example demonstrates using the generated AuthSome TypeScript client
 * with plugin composition.
 */

import { 
  AuthsomeClient, 
  socialClient, 
  twofaClient,
  AuthsomeError 
} from '../../../clients/typescript/src';

async function main() {
  // Initialize client with plugins
  const client = new AuthsomeClient({
    baseURL: 'http://localhost:8080',
    plugins: [
      socialClient(),
      twofaClient(),
    ]
  });

  console.log('AuthSome TypeScript Client Example\n');

  try {
    // Example 1: User Registration
    console.log('1. Registering new user...');
    const signUpResult = await client.signUp({
      email: 'test@example.com',
      password: 'SecurePassword123!',
      name: 'Test User'
    });
    console.log('✓ User registered:', signUpResult.user.email);
    console.log('✓ Session created:', signUpResult.session.id);

    // Store token for authenticated requests
    client.setToken(signUpResult.session.token);

    // Example 2: Get Current Session
    console.log('\n2. Fetching current session...');
    const sessionResult = await client.getSession();
    console.log('✓ Current user:', sessionResult.user.email);
    console.log('✓ Session expires:', sessionResult.session.expiresAt);

    // Example 3: Update User Profile
    console.log('\n3. Updating user profile...');
    const updateResult = await client.updateUser({
      name: 'Updated Test User'
    });
    console.log('✓ Profile updated:', updateResult.user.name);

    // Example 4: Social OAuth Plugin
    console.log('\n4. Using social OAuth plugin...');
    const socialPlugin = client.getPlugin<any>('social');
    if (socialPlugin) {
      const oauthUrl = await socialPlugin.signIn({
        provider: 'google',
        scopes: ['email', 'profile']
      });
      console.log('✓ OAuth URL generated:', oauthUrl.url);
    }

    // Example 5: List Devices
    console.log('\n5. Listing devices...');
    const devicesResult = await client.listDevices();
    console.log(`✓ Found ${devicesResult.devices.length} device(s)`);

    // Example 6: Sign Out
    console.log('\n6. Signing out...');
    await client.signOut();
    console.log('✓ Signed out successfully');

  } catch (error) {
    if (error instanceof AuthsomeError) {
      console.error('❌ API Error:', error.message);
      console.error('   Status Code:', error.statusCode);
      console.error('   Error Code:', error.code);
    } else {
      console.error('❌ Unexpected error:', error);
    }
    process.exit(1);
  }
}

// Run the example
main().then(() => {
  console.log('\n✓ Example completed successfully!');
  process.exit(0);
}).catch((error) => {
  console.error('\n❌ Example failed:', error);
  process.exit(1);
});

