import { Vulnerability } from "@/types/vulnerability.types"

export const MOCK_VULNS: Vulnerability[] = Array.from({ length: 18 }, (_, i) => {
   const severity = (["critical", "high", "medium", "low"] as const)[i % 4]
   // Simulate long content for edge case testing
   const isLong = i === 1 || i === 5
   // Simulate Dalfox finding
   const isDalfox = i % 3 === 1
   const source = isDalfox ? "dalfox" : i % 3 === 0 ? "nuclei" : "custom"

   return {
      id: i + 1,
      vulnType: isLong
         ? "Blind SQL Injection via User-Agent Header in /api/v1/auth/login endpoint resulting in Database Extraction (Time-Based)"
         : isDalfox
            ? "XSS (Reflected) in search parameter"
            : i % 3 === 0
               ? "SQL Injection in /api/login"
               : "Sensitive Data Exposure",
      severity,
      source,
      url: isLong
         ? "https://api.target-system.com/v1/endpoints/search?q=test&category=all&sort=desc&filter[active]=true&filter[role]=admin&session_id=abcdef1234567890&tracking_code=utm_source_google_campaign_spring_sale_2024&very_long_param_to_test_truncation=true"
         : `https://api.target-system.com/v1/endpoints/${i}`,
      isReviewed: i < 5,
      createdAt: new Date(Date.now() - i * 1000 * 60 * 60).toISOString(),
      updatedAt: new Date().toISOString(),
      description: isLong
         ? "The application appears to be vulnerable to SQL Injection. The input parameter 'id' is not properly sanitized before being used in a SQL query. This allows an attacker to manipulate the query structure, potentially accessing unauthorized data, modifying database contents, or executing administrative operations. The vulnerability was confirmed using a time-based blind injection technique where the server response was delayed by 5 seconds."
         : "The application appears to be vulnerable to SQL Injection.",
      cvssScore: severity === "critical" ? 9.8 : severity === "high" ? 7.5 : 5.0,
      rawOutput: {
         "curl-command": isLong
            ? `curl -X POST "https://api.target-system.com/v1/endpoints/${i}?token=very_long_token_string_that_goes_on_and_on" -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36" -H "Content-Type: application/json" -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c" -d '{"id": "1 OR SLEEP(5)", "nested": {\"deeply\": {\"nested\": \"value\"}}}'`
            : `curl -X POST https://api.target-system.com/v1/endpoints/${i} -d 'id=1 OR 1=1'`,
         // Dalfox fields
         payload: isDalfox ? '"><script>alert(1)</script>' : undefined,
         evidence: isDalfox ? 'GET /?q=\"><script>alert(1)</script> HTTP/1.1' : undefined,
         param: isDalfox ? "q" : undefined,
         method: "GET",

         // Nuclei fields
         request: !isDalfox
            ? (isLong
               ? `POST /v1/endpoints/${i}?token=very_long_token_string_that_goes_on_and_on HTTP/1.1\nHost: api.target-system.com\nUser-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36\nContent-Type: application/json\nAuthorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c\nAccept: */*\nX-Forwarded-For: 127.0.0.1\nConnection: keep-alive\n\n{\n  "id": "1' OR SLEEP(5)--",\n  "comment": "This is a very long payload to test how the code block handles wrapping and scrolling. It should handle it gracefully without breaking the layout.",\n  "payload": "UNION SELECT 1,2,3,4,5,6,7,8,9,user(),database(),version(),@@hostname,@@datadir,group_concat(table_name) FROM information_schema.tables WHERE table_schema=database()--"\n}`
               : `POST /v1/endpoints/${i} HTTP/1.1\nHost: api.target-system.com\nContent-Type: application/json\n\n{\n  "id": "1' OR '1'='1"\n}`)
            : undefined,

         response: !isDalfox
            ? (isLong
               ? `HTTP/1.1 500 Internal Server Error\nDate: Mon, 27 Jul 2024 12:28:53 GMT\nContent-Type: application/json; charset=utf-8\nContent-Length: 1204\nServer: nginx/1.18.0 (Ubuntu)\nX-Powered-By: Express\n\n{\n  "error": "Database Error",\n  "message": "Syntax error or access violation: 1064 You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near '' OR SLEEP(5)--' at line 1",\n  "stack": "Error: ER_PARSE_ERROR: You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near '' OR SLEEP(5)--' at line 1\\n    at Query.Sequence._packetToError (/app/node_modules/mysql/lib/protocol/sequences/Sequence.js:47:14)\\n    at Query.ErrorPacket (/app/node_modules/mysql/lib/protocol/sequences/Query.js:79:18)\\n    at Protocol._parsePacket (/app/node_modules/mysql/lib/protocol/Protocol.js:291:23)\\n    at Parser.write (/app/node_modules/mysql/lib/protocol/Parser.js:80:12)\\n    at Protocol.write (/app/node_modules/mysql/lib/protocol/Protocol.js:38:16)\\n    at Socket.<anonymous> (/app/node_modules/mysql/lib/Connection.js:88:28)\\n    at Socket.<anonymous> (/app/node_modules/mysql/lib/Connection.js:526:10)\\n    at Socket.emit (events.js:315:20)\\n    at addChunk (_stream_readable.js:309:12)\\n    at readableAddChunk (_stream_readable.js:284:9)"\n}`
               : `HTTP/1.1 200 OK\nContent-Type: application/json\n\n{\n  "id": 1,\n  "role": "admin",\n  "data": "sensitive"\n}`)
            : undefined,

         info: {
            description: isDalfox
               ? "Reflected Cross-Site Scripting (XSS) occurs when an application receives data in an HTTP request and includes that data within the immediate response in an unsafe way."
               : "SQL Injection detected via boolean-based blind technique.",
            classification: {
               "cwe-id": isDalfox ? ["CWE-79"] : ["CWE-89"],
               "cve-id": "CVE-2024-XXXX"
            },
            reference: [
               "https://owasp.org/www-community/attacks/SQL_Injection",
               "https://cwe.mitre.org/data/definitions/89.html",
               "https://portswigger.net/web-security/sql-injection",
               "https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html"
            ]
         }
      }
   }
})
