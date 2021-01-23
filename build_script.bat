set project_name=whatsapp-signal-notifier

mkdir build
cd build

mkdir %project_name%\appdata
copy ..\appdata\switch_to_signal_video.mp4 %project_name%\appdata\
copy ..\appdata\reply_text.txt %project_name%\appdata\

set GOOS=windows
set file_name=%project_name%\%project_name%.exe
go build -o %file_name% ..\
7z a -tzip %project_name%-windows %project_name%\
del %file_name%

set GOOS=darwin
set file_name=%project_name%\%project_name%
go build -o %file_name% ..\
7z a -tzip %project_name%-mac %project_name%\
del %file_name%

set GOOS=linux
set file_name=%project_name%\%project_name%
go build -o %file_name% ..\
7z a -tzip %project_name%-linux %project_name%\
del %file_name%

rmdir /S /Q %project_name%