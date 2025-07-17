# Frontend Developer Guide: Building the Metrics API Endpoint

## **Overview**
You need to create an API endpoint that accepts hardware metrics from the gswarm-sidecar and stores them in your Supabase database. The wallet address is extracted from the JWT token in the Authorization header.

## **Data Structure from gswarm-sidecar**

### **Incoming Request Format**
```json
{
  "node_id": "my-node-123",
  "timestamp": "2024-01-01T12:00:00Z",
  "metrics_type": "hardware",
  "data": {
    "cpu": {
      "percent": 45.2,
      "cores": 8,
      "load_avg": [1.2, 1.1, 1.0]
    },
    "ram": {
      "total": 17179869184,
      "used": 8589934592,
      "available": 8589934592,
      "usage_percent": 50.0,
      "swap_total": 4294967296,
      "swap_used": 1073741824,
      "swap_percent": 25.0
    },
    "gpu": [
      {
        "index": 0,
        "util_percent": 78.5,
        "temp_c": 65.0,
        "vram_used_mb": 6144,
        "vram_total_mb": 8192
      }
    ]
  }
}
```

### **Headers**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

## **JWT Token Structure**
The JWT token contains the wallet address in the payload. You'll need to decode it to extract the wallet address:

```json
{
  "sub": "user_id",
  "wallet_address": "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6",
  "iat": 1640995200,
  "exp": 1641081600
}
```

## **API Endpoint Implementation**

### **1. JWT Decoding Utility**

```typescript
// utils/jwt.ts
import jwt from 'jsonwebtoken'

interface JWTPayload {
  sub: string
  wallet_address: string
  iat: number
  exp: number
}

export function decodeJWT(token: string): JWTPayload | null {
  try {
    // Remove 'Bearer ' prefix if present
    const cleanToken = token.replace('Bearer ', '')

    // Decode without verification (since you're using service key)
    // In production, you might want to verify the signature
    const decoded = jwt.decode(cleanToken) as JWTPayload

    if (!decoded || !decoded.wallet_address) {
      return null
    }

    return decoded
  } catch (error) {
    console.error('JWT decode error:', error)
    return null
  }
}
```

### **2. Create the Endpoint**

```typescript
// pages/api/metrics.ts (Next.js) or app/api/metrics/route.ts (App Router)
import { createClient } from '@supabase/supabase-js'
import { NextApiRequest, NextApiResponse } from 'next'
import { decodeJWT } from '@/utils/jwt'

const supabase = createClient(
  process.env.SUPABASE_URL!,
  process.env.SUPABASE_SERVICE_KEY!
)

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' })
  }

  try {
    // Extract and validate JWT
    const authHeader = req.headers.authorization
    if (!authHeader) {
      return res.status(401).json({ error: 'Missing authorization header' })
    }

    const jwtPayload = decodeJWT(authHeader)
    if (!jwtPayload) {
      return res.status(401).json({ error: 'Invalid JWT token' })
    }

    const walletAddress = jwtPayload.wallet_address
    const { node_id, timestamp, metrics_type, data } = req.body

    // Validate required fields
    if (!node_id || !metrics_type || !data) {
      return res.status(400).json({ error: 'Missing required fields' })
    }

    // Store latest metrics (upsert)
    const { error: latestError } = await supabase
      .from('latest_metrics')
      .upsert({
        node_id,
        wallet_address: walletAddress,
        metrics_type,
        data,
        updated_at: new Date().toISOString()
      })

    if (latestError) {
      console.error('Error storing latest metrics:', latestError)
      return res.status(500).json({ error: 'Failed to store latest metrics' })
    }

    // Store historical metrics
    const { error: historicalError } = await supabase
      .from('historical_metrics')
      .insert({
        node_id,
        wallet_address: walletAddress,
        metrics_type,
        data
      })

    if (historicalError) {
      console.error('Error storing historical metrics:', historicalError)
      return res.status(500).json({ error: 'Failed to store historical metrics' })
    }

    res.status(200).json({ status: 'success' })
  } catch (error) {
    console.error('API error:', error)
    res.status(500).json({ error: 'Internal server error' })
  }
}
```

### **3. App Router Version (if using Next.js 13+)**

```typescript
// app/api/metrics/route.ts
import { createClient } from '@supabase/supabase-js'
import { NextRequest, NextResponse } from 'next/server'
import { decodeJWT } from '@/utils/jwt'

const supabase = createClient(
  process.env.SUPABASE_URL!,
  process.env.SUPABASE_SERVICE_KEY!
)

export async function POST(request: NextRequest) {
  try {
    // Extract and validate JWT
    const authHeader = request.headers.get('authorization')
    if (!authHeader) {
      return NextResponse.json(
        { error: 'Missing authorization header' },
        { status: 401 }
      )
    }

    const jwtPayload = decodeJWT(authHeader)
    if (!jwtPayload) {
      return NextResponse.json(
        { error: 'Invalid JWT token' },
        { status: 401 }
      )
    }

    const walletAddress = jwtPayload.wallet_address
    const body = await request.json()
    const { node_id, timestamp, metrics_type, data } = body

    // Validate required fields
    if (!node_id || !metrics_type || !data) {
      return NextResponse.json(
        { error: 'Missing required fields' },
        { status: 400 }
      )
    }

    // Store latest metrics (upsert)
    const { error: latestError } = await supabase
      .from('latest_metrics')
      .upsert({
        node_id,
        wallet_address: walletAddress,
        metrics_type,
        data,
        updated_at: new Date().toISOString()
      })

    if (latestError) {
      console.error('Error storing latest metrics:', latestError)
      return NextResponse.json(
        { error: 'Failed to store latest metrics' },
        { status: 500 }
      )
    }

    // Store historical metrics
    const { error: historicalError } = await supabase
      .from('historical_metrics')
      .insert({
        node_id,
        wallet_address: walletAddress,
        metrics_type,
        data
      })

    if (historicalError) {
      console.error('Error storing historical metrics:', historicalError)
      return NextResponse.json(
        { error: 'Failed to store historical metrics' },
        { status: 500 }
      )
    }

    return NextResponse.json({ status: 'success' })
  } catch (error) {
    console.error('API error:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}
```

## **Database Schema**

```sql
-- Latest metrics table
CREATE TABLE latest_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id TEXT NOT NULL,
    wallet_address TEXT NOT NULL,
    metrics_type TEXT NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(node_id, wallet_address, metrics_type)
);

-- Historical metrics table
CREATE TABLE historical_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id TEXT NOT NULL,
    wallet_address TEXT NOT NULL,
    metrics_type TEXT NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_latest_metrics_wallet ON latest_metrics(wallet_address);
CREATE INDEX idx_latest_metrics_node_wallet ON latest_metrics(node_id, wallet_address);
CREATE INDEX idx_historical_metrics_wallet ON historical_metrics(wallet_address);
CREATE INDEX idx_historical_metrics_node_wallet ON historical_metrics(node_id, wallet_address);
CREATE INDEX idx_historical_metrics_wallet_time ON historical_metrics(wallet_address, created_at DESC);
```

## **Environment Variables**

```bash
# .env.local
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_KEY=your-service-key-here
JWT_SECRET=your-jwt-secret-if-verifying-signatures
```

## **Testing the Endpoint**

### **Test with curl:**
```bash
curl -X POST http://localhost:3000/api/metrics \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{
    "node_id": "test-node-123",
    "timestamp": "2024-01-01T12:00:00Z",
    "metrics_type": "hardware",
    "data": {
      "cpu": {
        "percent": 45.2,
        "cores": 8,
        "load_avg": [1.2, 1.1, 1.0]
      },
      "ram": {
        "total_mb": 16384,
        "used_mb": 8192,
        "percent_used": 50.0
      }
    }
  }'
```

## **Error Handling**

The endpoint handles these scenarios:
- ✅ **Missing authorization header** (401 Unauthorized)
- ✅ **Invalid JWT token** (401 Unauthorized)
- ✅ **Missing required fields** (400 Bad Request)
- ✅ **Database errors** (500 Internal Server Error)
- ✅ **Invalid HTTP method** (405 Method Not Allowed)

## **Security Considerations**

1. **JWT Validation**: Consider verifying JWT signatures in production
2. **Rate Limiting**: Implement rate limiting per wallet address
3. **Input Validation**: Validate all incoming data
4. **Logging**: Log all requests for audit purposes

## **Next Steps**

1. **Deploy the endpoint** to your hosting platform
2. **Update the gswarm-sidecar config** with your endpoint URL
3. **Test with real data** from a gswarm node
4. **Monitor the logs** for any errors
5. **Create a dashboard** to view the metrics

The endpoint is now ready to receive and store hardware metrics with JWT-based wallet authentication! 🚀
