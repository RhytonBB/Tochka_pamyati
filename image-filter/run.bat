@echo off

echo Activating virtual environment...
call .\venv\Scripts\activate

echo Starting the image filter service...
uvicorn main:app --host %APP_HOST% --port %APP_PORT% --reload

pause
