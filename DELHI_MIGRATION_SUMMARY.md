# Delhi Data Migration Summary

## Overview

The backend service has been updated to support Delhi Metro and Bus transit data instead of Bangalore (BMTC) data. Both datasets have passed quality checks and are ready for use.

## Quality Check Results

### ✅ Delhi Metro (DMRC)
- **36 routes** covering all Delhi Metro lines
- **262 stations** with accurate coordinates
- **5,438 trips** with complete schedules
- **128,434 stop_times** entries
- **6,643 shape points** for route visualization
- **Quality Score**: EXCELLENT - No issues found

### ✅ Delhi Bus (DIMTS + DTC)
- **2,403 routes** (1,402 DIMTS + 1,001 DTC)
- **10,559 bus stops** with accurate coordinates
- **89,393 trips** with complete schedules
- **3,724,320 stop_times** entries
- **2,305,138 fare rules** for accurate fare calculation
- **Quality Score**: EXCELLENT - No issues found

## Changes Made

### 1. Fare Service Updates (`backend/internal/services/fare_service.go`)
- Added Delhi Metro fare rules (DMRC): ₹10 base fare, ₹2.5/km
- Added Delhi Bus fare rules (DIMTS/DTC): ₹5 base fare, ₹1.5/km
- Added `GetAgencyIDFromRoute()` method to dynamically get agency ID from route
- Updated `GetFareRulesForAgency()` to support DMRC, DIMTS, and DTC agencies

### 2. Configuration Updates (`backend/internal/config/config.go`)
- Changed default GTFS data path from Bangalore to Delhi Metro: `../DMRC_GTFS`

### 3. Service Updates
- **Route Planner**: Updated to use agency-specific fare rules
- **Route Boarding Service**: Updated to get agency ID from route
- **Journey Session Service**: Updated default agency to DIMTS
- **Fare Handler**: Updated to dynamically get agency ID from route

### 4. New Delhi Data Loader (`backend/cmd/loader-delhi/main.go`)
- Created specialized loader for Delhi data
- Supports loading metro only, bus only, or both datasets
- Uses GTFS aggregator to merge multiple feeds

## How to Load Delhi Data

### Option 1: Load Both Metro and Bus Data (Recommended)
```bash
cd backend
go run cmd/loader-delhi/main.go
```

This will load:
- Metro data from `../DMRC_GTFS`
- Bus data from `../GTFS (1)`
- Merge both feeds into a single database

### Option 2: Load Metro Only
```bash
cd backend
go run cmd/loader-delhi/main.go -metro-only
```

### Option 3: Load Bus Only
```bash
cd backend
go run cmd/loader-delhi/main.go -bus-only
```

### Option 4: Custom Paths
```bash
cd backend
go run cmd/loader-delhi/main.go -metro /path/to/metro -bus /path/to/bus
```

## Data Structure

### Agencies
- **DMRC**: Delhi Metro Rail Corporation (Metro)
- **DIMTS**: Delhi Integrated Multi-Modal Transit System Ltd. (Bus)
- **DTC**: Delhi Transport Corporation (Bus)

### Route Types
- **Metro**: Route type 1 (Metro/Subway)
- **Bus**: Route type 3 (Bus)

## Fare Calculation

The system now automatically uses the correct fare rules based on the agency:

- **DMRC (Metro)**: 
  - Base fare: ₹10
  - Per km: ₹2.5
  - No transfer fees within metro system

- **DIMTS/DTC (Bus)**:
  - Base fare: ₹5
  - Per km: ₹1.5
  - Transfer fee: ₹2
  - AC bus multiplier: 1.5x
  - Express bus multiplier: 1.2x

## API Usage

All existing API endpoints work the same way, but now use Delhi data:

### Journey Planning
```bash
POST /api/v1/journeys/plan?from_lat=28.6139&from_lon=77.2090&to_lat=28.5355&to_lon=77.3910
```

### Stop Search
```bash
GET /api/v1/stops/search?q=Connaught
```

### Route Search
```bash
GET /api/v1/routes/search?q=Yellow
```

### Fare Calculation
```bash
GET /api/v1/fares/calculate?route_id=5&from_stop_id=49&to_stop_id=50
```

The fare calculation automatically detects the agency (DMRC/DIMTS/DTC) and applies the correct fare rules.

## Testing

After loading the data, test the system:

1. **Check agencies**:
   ```bash
   curl http://localhost:8080/api/v1/stops/search?q=Metro
   ```

2. **Plan a journey**:
   ```bash
   curl -X POST "http://localhost:8080/api/v1/journeys/plan?from_lat=28.6139&from_lon=77.2090&to_lat=28.5355&to_lon=77.3910"
   ```

3. **Calculate fare**:
   ```bash
   curl "http://localhost:8080/api/v1/fares/calculate?route_id=5&from_stop_id=49&to_stop_id=50"
   ```

## Next Steps

1. **Load the data**: Run the Delhi loader to populate the database
2. **Test APIs**: Verify all endpoints work with Delhi data
3. **Update documentation**: Update API docs with Delhi-specific examples
4. **Monitor performance**: Check query performance with larger dataset

## Notes

- The Delhi bus dataset is significantly larger than Bangalore (3.7M stop_times vs 128K)
- Ensure PostgreSQL has sufficient resources for the larger dataset
- Consider indexing optimization for better query performance
- The fare rules can be further refined based on actual Delhi transit fare structure

---

**Migration Date**: $(date)
**Status**: ✅ Complete - Ready for testing

