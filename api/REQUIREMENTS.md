# Requirements: Core Backend API ()

## 1. Objective
Build a Node.js REST API running on port  backed by MongoDB. It manages saved API endpoint presets (), triggers the Python  to run benchmarks, persists historical results (), and computes before/after comparisons.

## 2. Tech Stack
- Node.js + Express
- MongoDB + Mongoose
-  + 

## 3. Database Models

### Collection 1: 
- : String, required, trimmed (e.g., Local Health Check).
- : String, required.
- : String, enum , default .
- : Map of String (or plain Object), default empty object.
- : Mixed/Object, default .
- : Enabled (, ).

### Collection 2: 
- : ObjectId referencing , required, **indexed**.
- : Object containing  (Number) and  (Number).
- : Object containing , , , , , ,  (all Numbers).
- : Map of Number (or plain Object).
- : Enabled ().

## 4. API Endpoints

1. ****
   - Creates and saves a new target preset. Returns  with the created document.
2. ****
   - Lists all saved targets, sorted newest first.
3. ****
   - Deletes a target and also deletes all  documents associated with that .
4. ****
   - Accepts  in the request body.
   - Looks up the  by .
   - Calls the Python engine () with the target's details and the run config.
   - Saves the returned metrics into the  collection linked to .
   - Returns  with the saved  document.
5. ****
   - Returns all historical runs for a specific target, sorted newest first.
6. ****
   - Fetches both runs from MongoDB.
   - Computes the percentage change from  to  for: , , , and .
   - Returns  where  contains the percentage deltas (e.g., ).

## 5. Edge Cases to Handle
- Invalid MongoDB ObjectIds in  or query params should return , not crash Express.
- Non-existent  or  should return .
- If the Python engine is offline or times out, return  with a clear error message.
- Guard against division by zero in  (e.g., when  is ).

## 6. Implementation Hints
- Enable  middleware early so the React app on port  can talk to port .
- Node 18+ has native  built in—you don't need  to call the Python engine.
- Put the Python engine URL () and MongoDB URI in a  file.
- Formula for percentage change: . Remember: for latency (, ), a negative percentage is an **improvement**; for , a positive percentage is an **improvement**.
- Create a simple  route on this same Node server—it gives you a safe, fast local endpoint to benchmark against!

## 7. Definition of Done
- You can create a target pointing to  via .
- Triggering  twice saves two documents in MongoDB.
- Calling  with those two IDs returns accurate percentage deltas.
